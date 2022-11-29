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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	kafkaMirrorTopicNameWithPrefix              = "us_orders"
	scenarioStateKafkaMirrorTopicHasBeenStopped = "The Kafka Mirror Topic has been stopped"
)

var createKafkaMirrorTopicWithPrefixPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors", destinationClusterId, clusterLinkName)
var readKafkaMirrorTopicWithPrefixPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicNameWithPrefix)
var deleteKafkaMirrorTopicWithPrefixPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics/%s", destinationClusterId, kafkaMirrorTopicNameWithPrefix)
var stopKafkaMirrorTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors:failover", destinationClusterId, clusterLinkName)

func TestAccKafkaMirrorTopicWithPrefix(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockKafkaMirrorTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockKafkaMirrorTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createClusterLinkResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/with_prefix/create_kafka_mirror_topic.json")
	createClusterLinkStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaMirrorTopicWithPrefixPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillReturn(
			string(createClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createClusterLinkStub)

	readCreatedClusterLinkResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/with_prefix/read_created_kafka_mirror_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaMirrorTopicWithPrefixPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillReturn(
			string(readCreatedClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateTopicStub := wiremock.Post(wiremock.URLPathEqualTo(stopKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaMirrorTopicHasBeenStopped).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(updateTopicStub)

	readUpdatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/with_prefix/read_stopped_kafka_mirror_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaMirrorTopicWithPrefixPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenStopped).
		WillReturn(
			string(readUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaMirrorTopicWithPrefixPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteClusterLinkStub := wiremock.Delete(wiremock.URLPathEqualTo(deleteKafkaMirrorTopicWithPrefixPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenStopped).
		WillSetStateTo(scenarioStateKafkaMirrorTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterLinkStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_KAFKA_API_KEY", destinationClusterApiKey)
	_ = os.Setenv("IMPORT_KAFKA_API_SECRET", destinationClusterApiSecret)
	_ = os.Setenv("IMPORT_KAFKA_REST_ENDPOINT", mockKafkaMirrorTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_KAFKA_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaMirrorTopicDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaMirrorTopicWithPrefixConfig(confluentCloudBaseUrl, mockKafkaMirrorTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaMirrorTopicExists(fullKafkaMirrorTopicResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "mirror_topic_name", kafkaMirrorTopicNameWithPrefix),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "status", stateActive),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.0.topic_name", kafkaMirrorTopicName),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.0.link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.%", "3"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.id", destinationClusterId),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.rest_endpoint", mockKafkaMirrorTestServerUrl),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.key", destinationClusterApiKey),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.secret", destinationClusterApiSecret),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicNameWithPrefix)),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "%", numberOfKafkaMirrorTopicResourceAttributes),
				),
			},
			{
				Config: testAccCheckUpdatedKafkaMirrorTopicWithPrefixConfig(confluentCloudBaseUrl, mockKafkaMirrorTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaMirrorTopicExists(fullKafkaMirrorTopicResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "mirror_topic_name", kafkaMirrorTopicNameWithPrefix),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "status", stateStopped),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "source_kafka_topic.0.topic_name", kafkaMirrorTopicName),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.0.%", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "cluster_link.0.link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.%", "3"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.id", destinationClusterId),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.rest_endpoint", mockKafkaMirrorTestServerUrl),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.key", destinationClusterApiKey),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "kafka_cluster.0.credentials.0.secret", destinationClusterApiSecret),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicNameWithPrefix)),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "%", numberOfKafkaMirrorTopicResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullKafkaMirrorTopicResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterLinkStub, fmt.Sprintf("POST %s", createClusterLinkDestinationOutboundPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteClusterLinkStub, fmt.Sprintf("DELETE %s", readClusterLinkDestinationOutboundPath), expectedCountOne)
}

func testAccCheckKafkaMirrorTopicWithPrefixConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_kafka_mirror_topic" "%s" {
      mirror_topic_name = "%s"
	  source_kafka_topic {
        topic_name = "%s"
      }
      cluster_link {
        link_name = "%s"
      }
      kafka_cluster {
        id = "%s"
        rest_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }
	}
	`, confluentCloudBaseUrl, kafkaMirrorTopicResourceLabel,
		kafkaMirrorTopicNameWithPrefix, kafkaMirrorTopicName, clusterLinkName,
		destinationClusterId, mockServerUrl, destinationClusterApiKey, destinationClusterApiSecret)
}

func testAccCheckUpdatedKafkaMirrorTopicWithPrefixConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_kafka_mirror_topic" "%s" {
      status = "%s"
      mirror_topic_name = "%s"
	  source_kafka_topic {
        topic_name = "%s"
      }
      cluster_link {
        link_name = "%s"
      }
      kafka_cluster {
        id = "%s"
        rest_endpoint = "%s"
        credentials {
		  key = "%s"
		  secret = "%s"
	    }
      }
	}
	`, confluentCloudBaseUrl, kafkaMirrorTopicResourceLabel, stateFailedOver,
		kafkaMirrorTopicNameWithPrefix, kafkaMirrorTopicName, clusterLinkName,
		destinationClusterId, mockServerUrl, destinationClusterApiKey, destinationClusterApiSecret)
}
