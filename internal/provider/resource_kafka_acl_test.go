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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateAclHasBeenCreated = "A new ACL has been just created"
	scenarioStateAclHasBeenDeleted = "The ACL has been deleted"
	aclScenarioName                = "confluent_kafka_acl Resource Lifecycle"
	aclPatternType                 = "LITERAL"
	aclResourceName                = "kafka-cluster"
	aclPrincipalWithIntegerId      = "User:732363"
	aclPrincipalWithResourceId     = "User:sa-abc123"
	aclHost                        = "*"
	aclOperation                   = "READ"
	aclPermission                  = "ALLOW"
	aclResourceType                = "CLUSTER"
	aclResourceLabel               = "test_acl_resource_label"
)

var fullAclResourceLabel = fmt.Sprintf("confluent_kafka_acl.%s", aclResourceLabel)
var createKafkaAclPath = fmt.Sprintf("/kafka/v3/clusters/%s/acls", clusterId)
var readServiceAccountsPath = "/service_accounts"
var readKafkaAclPath = fmt.Sprintf("/kafka/v3/clusters/%s/acls?host=%s&operation=%s&pattern_type=%s&permission=%s&principal=%s&resource_name=%s&resource_type=%s", clusterId, aclHost, aclOperation, aclPatternType, aclPermission, aclPrincipalWithIntegerId, aclResourceName, aclResourceType)

// TODO: APIF-1990
var mockAclTestServerUrl = ""

func TestAccAcls(t *testing.T) {
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
				Config: testAccCheckAclConfig(confluentCloudBaseUrl, mockAclTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAclExists(fullAclResourceLabel),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "id", fmt.Sprintf("%s/%s#%s#%s#%s#%s#%s#%s", clusterId, aclResourceType, aclResourceName, aclPatternType, aclPrincipalWithResourceId, aclHost, aclOperation, aclPermission)),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "resource_type", aclResourceType),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "resource_name", aclResourceName),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "pattern_type", aclPatternType),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "principal", aclPrincipalWithResourceId),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "host", aclHost),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "operation", aclOperation),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "permission", aclPermission),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullAclResourceLabel, "credentials.0.secret", kafkaApiSecret),
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

func testAccCheckAclDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client).kafkaRestClientFactory.CreateKafkaRestClient(mockAclTestServerUrl, clusterId, kafkaApiKey, kafkaApiSecret, false)
	// Loop through the resources in state, verifying each ACL is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_acl" {
			continue
		}
		deletedAclId := rs.Primary.ID
		aclList, _, err := c.apiClient.ACLV3Api.GetKafkaAcls(c.apiContext(context.Background()), clusterId).ResourceType(aclResourceType).ResourceName(aclResourceName).PatternType(aclPatternType).Principal(aclPrincipalWithIntegerId).Host(aclHost).Operation(aclOperation).Permission(aclPermission).Execute()

		if len(aclList.Data) == 0 {
			return nil
		} else if err == nil && deletedAclId != "" {
			// Otherwise return the error
			if deletedAclId == rs.Primary.ID {
				return fmt.Errorf("ACL (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAclConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_kafka_acl" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	  resource_type = "%s"
	  resource_name = "%s"
	  pattern_type = "%s"
	  principal = "%s"
	  host = "*"
	  operation = "%s"
	  permission = "%s"

	  rest_endpoint = "%s"

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, aclResourceLabel, clusterId, aclResourceType, aclResourceName, aclPatternType, aclPrincipalWithResourceId,
		aclOperation, aclPermission, mockServerUrl, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckAclExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s ACL has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s ACL", n)
		}

		return nil
	}
}
