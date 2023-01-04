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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccAclsWithEnhancedProviderBlock2(t *testing.T) {
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

	mockAclTestServerUrl = fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	confluentCloudBaseUrl := mockAclTestServerUrl
	wiremockClient := wiremock.NewClient(mockAclTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readServiceAccountsResponse, _ := ioutil.ReadFile("../testdata/kafka_acl/read_service_accounts.json")
	readServiceAccountsStub := wiremock.Get(wiremock.URLPathEqualTo(readServiceAccountsPath)).
		InScenario(aclScenarioName).
		WillReturn(
			string(readServiceAccountsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readServiceAccountsStub)

	createAclStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaAclPath)).
		InScenario(aclScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAclHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAclStub)
	readCreatedAclResponse, _ := ioutil.ReadFile("../testdata/kafka_acl/search_created_kafka_acls.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(createKafkaAclPath)).
		WithQueryParam("host", wiremock.EqualTo(aclHost)).
		WithQueryParam("operation", wiremock.EqualTo(aclOperation)).
		WithQueryParam("pattern_type", wiremock.EqualTo(aclPatternType)).
		WithQueryParam("permission", wiremock.EqualTo(aclPermission)).
		WithQueryParam("principal", wiremock.EqualTo(aclPrincipalWithIntegerId)).
		WithQueryParam("resource_name", wiremock.EqualTo(aclResourceName)).
		WithQueryParam("resource_type", wiremock.EqualTo(aclResourceType)).
		InScenario(aclScenarioName).
		WhenScenarioStateIs(scenarioStateAclHasBeenCreated).
		WillReturn(
			string(readCreatedAclResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEmptyAclResponse, _ := ioutil.ReadFile("../testdata/kafka_acl/search_deleted_kafka_acls.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(createKafkaAclPath)).
		WithQueryParam("host", wiremock.EqualTo(aclHost)).
		WithQueryParam("operation", wiremock.EqualTo(aclOperation)).
		WithQueryParam("pattern_type", wiremock.EqualTo(aclPatternType)).
		WithQueryParam("permission", wiremock.EqualTo(aclPermission)).
		WithQueryParam("principal", wiremock.EqualTo(aclPrincipalWithIntegerId)).
		WithQueryParam("resource_name", wiremock.EqualTo(aclResourceName)).
		WithQueryParam("resource_type", wiremock.EqualTo(aclResourceType)).
		InScenario(aclScenarioName).
		WhenScenarioStateIs(scenarioStateAclHasBeenDeleted).
		WillReturn(
			string(readEmptyAclResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAclResponse, _ := ioutil.ReadFile("../testdata/kafka_acl/delete_kafka_acls.json")
	deleteAclStub := wiremock.Delete(wiremock.URLPathEqualTo(createKafkaAclPath)).
		WithQueryParam("host", wiremock.EqualTo(aclHost)).
		WithQueryParam("operation", wiremock.EqualTo(aclOperation)).
		WithQueryParam("pattern_type", wiremock.EqualTo(aclPatternType)).
		WithQueryParam("permission", wiremock.EqualTo(aclPermission)).
		WithQueryParam("principal", wiremock.EqualTo(aclPrincipalWithIntegerId)).
		WithQueryParam("resource_name", wiremock.EqualTo(aclResourceName)).
		WithQueryParam("resource_type", wiremock.EqualTo(aclResourceType)).
		InScenario(aclScenarioName).
		WhenScenarioStateIs(scenarioStateAclHasBeenCreated).
		WillSetStateTo(scenarioStateAclHasBeenDeleted).
		WillReturn(
			string(readDeletedAclResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteAclStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_KAFKA_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_KAFKA_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_KAFKA_REST_ENDPOINT", mockAclTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_KAFKA_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAclDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAclConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockAclTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAclExists(fullAclResourceLabel),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "kafka_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullAclResourceLabel, "kafka_cluster.0.id"),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "id", fmt.Sprintf("%s/%s#%s#%s#%s#%s#%s#%s", clusterId, aclResourceType, aclResourceName, aclPatternType, aclPrincipalWithResourceId, aclHost, aclOperation, aclPermission)),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "resource_type", aclResourceType),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "resource_name", aclResourceName),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "pattern_type", aclPatternType),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "principal", aclPrincipalWithResourceId),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "host", aclHost),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "operation", aclOperation),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "permission", aclPermission),
					resource.TestCheckNoResourceAttr(fullAclResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullAclResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullAclResourceLabel, "credentials.0.secret"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullAclResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createAclStub, fmt.Sprintf("POST %s", createKafkaAclPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAclStub, fmt.Sprintf("DELETE %s", readKafkaAclPath), expectedCountOne)
}

func testAccCheckAclConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  kafka_api_key = "%s"
	  kafka_api_secret = "%s"
	  kafka_rest_endpoint = "%s"
	  kafka_id = "%s"
    }
	resource "confluent_kafka_acl" "%s" {
	  resource_type = "%s"
	  resource_name = "%s"
	  pattern_type = "%s"
	  principal = "%s"
	  host = "*"
	  operation = "%s"
	  permission = "%s"
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, clusterId, aclResourceLabel, aclResourceType, aclResourceName, aclPatternType, aclPrincipalWithResourceId,
		aclOperation, aclPermission)
}
