// Copyright 2021 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"context"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	schemaDataSourceScenarioName = "confluent_schema Data Source Lifecycle"
)

var fullSchemaDataSourceLabel = fmt.Sprintf("data.confluent_schema.%s", testSchemaResourceLabel)

func TestAccDataSourceSchema(t *testing.T) {
	containerPort := "8080"
	containerPortTcp := fmt.Sprintf("%s/tcp", containerPort)
	ctx := context.Background()
	listeningPort := wait.ForListeningPort(nat.Port(containerPortTcp))
	req := testcontainers.ContainerRequest{
		Image:        "rodolpheche/wiremock",
		ExposedPorts: []string{containerPortTcp},
		WaitingFor:   listeningPort,
	}
	wiremockContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	// nolint:errcheck
	defer wiremockContainer.Terminate(ctx)

	host, err := wiremockContainer.Host(ctx)
	require.NoError(t, err)

	wiremockHttpMappedPort, err := wiremockContainer.MappedPort(ctx, nat.Port(containerPort))
	require.NoError(t, err)

	mockSchemaTestServerUrl = fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedSchemasResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schema/read_schemas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemaDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "id", fmt.Sprintf("%s/%s/%d", testStreamGovernanceClusterId, testSubjectName, testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "format", testFormat),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema", testSchemaContent),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "version", strconv.Itoa(testSchemaVersion)),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_identifier", strconv.Itoa(testSchemaIdentifier)),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.#", "2"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.0.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.0.name", testFirstSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.0.subject_name", testFirstSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.0.version", strconv.Itoa(testFirstSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.1.%", "3"),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.1.name", testSecondSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.1.subject_name", testSecondSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "schema_reference.1.version", strconv.Itoa(testSecondSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemaDataSourceLabel, "%", strconv.Itoa(testNumberOfSchemaRegistrySchemaResourceAttributes)),
				),
			},
		},
	})
}

func testAccCheckSchemaDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_schema" "%s" {
	  schema_registry_cluster {
        id = "%s"
      }
      rest_endpoint = "%s"
      credentials {
        key = "%s"
        secret = "%s"
	  }
	  subject_name = "%s"
	  schema_identifier = %d
	}
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSubjectName, testSchemaIdentifier)
}
