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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterLinkBidirectionalOutbound(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockClusterLinkTestServerUrl := wiremockContainer.URI
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
	_ = os.Setenv("IMPORT_LOCAL_KAFKA_REST_ENDPOINT", mockClusterLinkTestServerUrl)
	_ = os.Setenv("IMPORT_LOCAL_KAFKA_API_KEY", sourceClusterApiKey)
	_ = os.Setenv("IMPORT_LOCAL_KAFKA_API_SECRET", sourceClusterApiSecret)
	_ = os.Setenv("IMPORT_REMOTE_KAFKA_BOOTSTRAP_ENDPOINT", destinationClusterBootstrapEndpoint)
	_ = os.Setenv("IMPORT_REMOTE_KAFKA_API_KEY", destinationClusterApiKey)
	_ = os.Setenv("IMPORT_REMOTE_KAFKA_API_SECRET", destinationClusterApiSecret)
	defer func() {
		_ = os.Unsetenv("IMPORT_LOCAL_KAFKA_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_LOCAL_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_LOCAL_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_REMOTE_KAFKA_BOOTSTRAP_ENDPOINT")
		_ = os.Unsetenv("IMPORT_REMOTE_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_REMOTE_KAFKA_API_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckClusterLinkSourceDestroy(s, mockClusterLinkTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterLinkBidirectionalOutboundConfig(confluentCloudBaseUrl, mockClusterLinkTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterLinkExists(fullClusterLinkResourceLabel),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_mode", linkModeBidirectional),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "connection_mode", connectionModeOutbound),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.id", sourceClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.rest_endpoint", mockClusterLinkTestServerUrl),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.bootstrap_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.credentials.0.key", sourceClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "local_kafka_cluster.0.credentials.0.secret", sourceClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.id", destinationClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.rest_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.bootstrap_endpoint", destinationClusterBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.credentials.0.key", destinationClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "remote_kafka_cluster.0.credentials.0.secret", destinationClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "id", fmt.Sprintf("%s/%s", sourceClusterId, clusterLinkName)),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "cluster_link_id", "qz0HDEV-Qz2B5aPFpcWQJQ"),
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
					sourceClusterId := resources[fullClusterLinkResourceLabel].Primary.Attributes["local_kafka_cluster.0.id"]
					destinationClusterId := resources[fullClusterLinkResourceLabel].Primary.Attributes["remote_kafka_cluster.0.id"]
					return linkName + "/" + linkMode + "/" + connectionMode + "/" + sourceClusterId + "/" + destinationClusterId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterLinkStub, fmt.Sprintf("POST %s", createClusterLinkSourceOutboundPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterLinkStub, fmt.Sprintf("DELETE %s", readClusterLinkSourceOutboundPath), expectedCountOne)
}

func testAccCheckClusterLinkBidirectionalOutboundConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_cluster_link" "%s" {
	  link_name = "%s"
      link_mode = "%s"
      connection_mode = "%s"
	  local_kafka_cluster {
        id = "%s"
        rest_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }

	  remote_kafka_cluster {
        id = "%s"
        bootstrap_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }
	}
	`, confluentCloudBaseUrl, clusterLinkResourceLabel,
		clusterLinkName, linkModeBidirectional, connectionModeOutbound,
		sourceClusterId, mockServerUrl, sourceClusterApiKey, sourceClusterApiSecret,
		destinationClusterId, destinationClusterBootstrapEndpoint, destinationClusterApiKey, destinationClusterApiSecret)
}
