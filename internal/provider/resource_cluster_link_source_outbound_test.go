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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var createClusterLinkSourceOutboundPath = fmt.Sprintf("/kafka/v3/clusters/%s/links", sourceClusterId)
var readClusterLinkSourceOutboundPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s", sourceClusterId, clusterLinkName)
var readClusterLinkSourceOutboundConfigPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/configs", sourceClusterId, clusterLinkName)

func TestAccClusterLinkSourceOutbound(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockClusterLinkTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockClusterLinkTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createClusterLinkResponse, _ := ioutil.ReadFile("../testdata/cluster_link/create_cluster_link.json")
	createClusterLinkStub := wiremock.Post(wiremock.URLPathEqualTo(createClusterLinkSourceOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateClusterLinkHasBeenCreated).
		WillReturn(
			string(createClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createClusterLinkStub)

	readCreatedClusterLinkResponse, _ := ioutil.ReadFile("../testdata/cluster_link/read_created_cluster_link_destination.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenCreated).
		WillReturn(
			string(readCreatedClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedClusterLinkConfigResponse, _ := ioutil.ReadFile("../testdata/cluster_link/read_created_cluster_link_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundConfigPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenCreated).
		WillReturn(
			string(readCreatedClusterLinkConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteClusterLinkStub := wiremock.Delete(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenCreated).
		WillSetStateTo(scenarioStateClusterLinkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterLinkStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_REST_ENDPOINT", mockClusterLinkTestServerUrl)
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_API_KEY", sourceClusterApiKey)
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_API_SECRET", sourceClusterApiSecret)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_BOOTSTRAP_ENDPOINT", destinationClusterBootstrapEndpoint)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_API_KEY", destinationClusterApiKey)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_API_SECRET", destinationClusterApiSecret)
	defer func() {
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_BOOTSTRAP_ENDPOINT")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_API_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterLinkSourceDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterLinkSourceOutboundConfig(confluentCloudBaseUrl, mockClusterLinkTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterLinkExists(fullClusterLinkResourceLabel),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_mode", linkModeSource),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "connection_mode", connectionModeOutbound),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.id", sourceClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.rest_endpoint", mockClusterLinkTestServerUrl),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.bootstrap_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.key", sourceClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.secret", sourceClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.id", destinationClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.rest_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.bootstrap_endpoint", destinationClusterBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.key", destinationClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.secret", destinationClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "id", fmt.Sprintf("%s/%s", sourceClusterId, clusterLinkName)),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "%", numberOfClusterLinkResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullClusterLinkResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					linkName := resources[fullClusterLinkResourceLabel].Primary.Attributes["link_name"]
					linkMode := resources[fullClusterLinkResourceLabel].Primary.Attributes["link_mode"]
					connectionMode := resources[fullClusterLinkResourceLabel].Primary.Attributes["connection_mode"]
					sourceClusterId := resources[fullClusterLinkResourceLabel].Primary.Attributes["source_kafka_cluster.0.id"]
					destinationClusterId := resources[fullClusterLinkResourceLabel].Primary.Attributes["destination_kafka_cluster.0.id"]
					return linkName + "/" + linkMode + "/" + connectionMode + "/" + sourceClusterId + "/" + destinationClusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterLinkStub, fmt.Sprintf("POST %s", createClusterLinkSourceOutboundPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterLinkStub, fmt.Sprintf("DELETE %s", readClusterLinkSourceOutboundPath), expectedCountOne)
}

func testAccCheckClusterLinkSourceOutboundConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_cluster_link" "%s" {
	  link_name = "%s"
      link_mode = "%s"
      connection_mode = "%s"
	  source_kafka_cluster {
        id = "%s"
        rest_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }

	  destination_kafka_cluster {
        id = "%s"
        bootstrap_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }
	}
	`, confluentCloudBaseUrl, clusterLinkResourceLabel,
		clusterLinkName, linkModeSource, connectionModeOutbound,
		sourceClusterId, mockServerUrl, sourceClusterApiKey, sourceClusterApiSecret,
		destinationClusterId, destinationClusterBootstrapEndpoint, destinationClusterApiKey, destinationClusterApiSecret)
}

func testAccCheckClusterLinkSourceDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client).kafkaRestClientFactory.CreateKafkaRestClient(mockClusterLinkTestServerUrl, sourceClusterId, sourceClusterApiKey, sourceClusterApiSecret, false)
	// Loop through the resources in state, verifying each Cluster Link is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_cluster_link" {
			continue
		}
		deletedClusterLinkId := rs.Primary.ID
		_, response, err := c.apiClient.ClusterLinkingV3Api.GetKafkaLink(c.apiContext(context.Background()), sourceClusterId, clusterLinkName).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedClusterLinkId != "" {
			// Otherwise return the error
			if deletedClusterLinkId == rs.Primary.ID {
				return fmt.Errorf("topic (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
