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
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	schemasDataSourceScenarioName = "confluent_schemas Data Source Lifecycle"
	fullSchemasDataSourceLabel    = "data.confluent_schemas.all_schemas"
	testSchemasDataSourceLabel    = "all_schemas"
	testSchemasSubjectName        = "some_record"
	testSchemasSomeRecordV1       = `
syntax = "proto3";
package examples;
message SomeRecord {
	string value1 = 1;
}
`
	testSchemasSomeRecordV2 = `
syntax = "proto3";
package examples;
message SomeRecord {
  string value1 = 1;
  string value2 = 2;
}
`
)

func TestAccDataSourceSchemas(t *testing.T) {
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

	readCreatedSchemasResponse, _ := ioutil.ReadFile("../testdata/schema_registry_schemas/read_some_record_schemas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemasPath)).
		InScenario(schemasDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSchemasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemasDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemasDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.#", "2"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.version", "1"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.format", "PROTOBUF"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.subject_name", "some_record"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.schema_identifier", "100001"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.schema_reference.#", "0"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.0.schema", strings.TrimLeft(testSchemasSomeRecordV1, "\n")),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.version", "2"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.format", "PROTOBUF"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.subject_name", "some_record"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_identifier", "100002"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.#", "2"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.0.%", "3"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.0.name", testFirstSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.0.subject_name", testFirstSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.0.version", strconv.Itoa(testFirstSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.1.%", "3"),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.1.name", testSecondSchemaReferenceDisplayName),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.1.subject_name", testSecondSchemaReferenceSubject),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema_reference.1.version", strconv.Itoa(testSecondSchemaReferenceVersion)),
					resource.TestCheckResourceAttr(fullSchemasDataSourceLabel, "schemas.1.schema", strings.TrimLeft(testSchemasSomeRecordV2, "\n")),
				),
			},
		},
	})
}

func testAccCheckSchemasDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_schemas" "%s" {
	  schema_registry_cluster {
        id = "%s"
      }
      rest_endpoint = "%s"
      credentials {
        key = "%s"
        secret = "%s"
	  }
	  filter {
		subject_prefix = "%s"
		latest_only = false
		deleted = true
	  }
	}
	`, confluentCloudBaseUrl, testSchemasDataSourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSchemasSubjectName)
}
