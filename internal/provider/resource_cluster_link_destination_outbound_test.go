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

const (
	scenarioStateClusterLinkHasBeenCreated = "A new cluster link has been just created"
	scenarioStateClusterLinkHasBeenDeleted = "The cluster link has been deleted"
	clusterLinkScenarioName                = "confluent_cluster_link Resource Lifecycle"
	sourceClusterId                        = "lkc-nv0zqv"
	sourceClusterRestEndpoint              = "https://pkc-pgq85.us-west-2.aws.confluent.cloud:443"
	sourceClusterBootstrapEndpoint         = "SASL_SSL://pkc-pgq85.us-west-2.aws.confluent.cloud:9092"
	sourceClusterApiKey                    = "sourceClusterApiKey"
	sourceClusterApiSecret                 = "sourceClusterApiSecret"
	destinationClusterId                   = "lkc-81knqq"
	// mockServerUrl will be used instead for TestAccClusterLinkDestination test
	destinationClusterRestEndpoint        = "https://pkc-3588w.us-east-1.aws.confluent.cloud:443"
	destinationClusterBootstrapEndpoint   = "SASL_SSL://pkc-3588w.us-east-1.aws.confluent.cloud:9092"
	destinationClusterApiKey              = "destinationClusterApiKey"
	destinationClusterApiSecret           = "destinationClusterApiSecret"
	clusterLinkName                       = "ui-test"
	clusterLinkMode                       = "DESTINATION"
	clusterLinkConnectionMode             = "OUTBOUND"
	clusterLinkResourceLabel              = "test_cluster_link_resource_label"
	numberOfClusterLinkResourceAttributes = "6"
)

var fullClusterLinkResourceLabel = fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel)

var createClusterLinkDestinationOutboundPath = fmt.Sprintf("/kafka/v3/clusters/%s/links", destinationClusterId)
var readClusterLinkDestinationOutboundPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s", destinationClusterId, clusterLinkName)

//// TODO: APIF-1990
var mockClusterLinkTestServerUrl = ""

func TestAccClusterLinkDestinationOutbound(t *testing.T) {
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
	createClusterLinkStub := wiremock.Post(wiremock.URLPathEqualTo(createClusterLinkDestinationOutboundPath)).
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
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkDestinationOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenCreated).
		WillReturn(
			string(readCreatedClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkDestinationOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(scenarioStateClusterLinkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteClusterLinkStub := wiremock.Delete(wiremock.URLPathEqualTo(readClusterLinkDestinationOutboundPath)).
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
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_BOOTSTRAP_ENDPOINT", sourceClusterBootstrapEndpoint)
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_API_KEY", sourceClusterApiKey)
	_ = os.Setenv("IMPORT_SOURCE_KAFKA_API_SECRET", sourceClusterApiSecret)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_REST_ENDPOINT", mockClusterLinkTestServerUrl)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_API_KEY", destinationClusterApiKey)
	_ = os.Setenv("IMPORT_DESTINATION_KAFKA_API_SECRET", destinationClusterApiSecret)
	defer func() {
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_BOOTSTRAP_ENDPOINT")
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_SOURCE_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_DESTINATION_KAFKA_API_SECRET")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterLinkDestinationDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterLinkDestinationOutboundConfig(confluentCloudBaseUrl, mockClusterLinkTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterLinkExists(fullClusterLinkResourceLabel),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "link_mode", clusterLinkMode),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "connection_mode", connectionModeOutbound),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.id", sourceClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.rest_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.bootstrap_endpoint", sourceClusterBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.key", sourceClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "source_kafka_cluster.0.credentials.0.secret", sourceClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.%", "4"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.id", destinationClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.rest_endpoint", mockClusterLinkTestServerUrl),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.bootstrap_endpoint", ""),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.key", destinationClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "destination_kafka_cluster.0.credentials.0.secret", destinationClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkResourceLabel, "id", fmt.Sprintf("%s/%s", destinationClusterId, clusterLinkName)),
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

	checkStubCount(t, wiremockClient, createClusterLinkStub, fmt.Sprintf("POST %s", createClusterLinkDestinationOutboundPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterLinkStub, fmt.Sprintf("DELETE %s", readClusterLinkDestinationOutboundPath), expectedCountOne)
}

func testAccCheckClusterLinkDestinationOutboundConfig(confluentCloudBaseUrl, mockServerUrl string) string {
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
        bootstrap_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }

	  destination_kafka_cluster {
        id = "%s"
        rest_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }
	}
	`, confluentCloudBaseUrl, clusterLinkResourceLabel,
		clusterLinkName, linkModeDestination, connectionModeOutbound,
		sourceClusterId, sourceClusterBootstrapEndpoint, sourceClusterApiKey, sourceClusterApiSecret,
		destinationClusterId, mockServerUrl, destinationClusterApiKey, destinationClusterApiSecret)
}
