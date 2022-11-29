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
	scenarioStateKafkaMirrorTopicHasBeenCreated = "A new Kafka Mirror Topic has been just created"
	scenarioStateKafkaMirrorTopicHasBeenPaused  = "The Kafka Mirror Topic has been paused"
	scenarioStateKafkaMirrorTopicHasBeenDeleted = "The Kafka Mirror Topic has been deleted"
	kafkaMirrorTopicScenarioName                = "confluent_cluster_link Resource Lifecycle"
	kafkaMirrorTopicResourceLabel               = "test_kafka_mirror_topic_resource_label"
	kafkaMirrorTopicName                        = "orders"
	numberOfKafkaMirrorTopicResourceAttributes  = "6"
)

var fullKafkaMirrorTopicResourceLabel = fmt.Sprintf("confluent_kafka_mirror_topic.%s", kafkaMirrorTopicResourceLabel)

var createKafkaMirrorTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors", destinationClusterId, clusterLinkName)
var readKafkaMirrorTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicName)
var deleteKafkaMirrorTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics/%s", destinationClusterId, kafkaMirrorTopicName)
var pauseKafkaMirrorTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/links/%s/mirrors:pause", destinationClusterId, clusterLinkName)

//// TODO: APIF-1990
var mockKafkaMirrorTestServerUrl = ""

func TestAccKafkaMirrorTopic(t *testing.T) {
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
	createClusterLinkResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/regular/create_kafka_mirror_topic.json")
	createClusterLinkStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillReturn(
			string(createClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createClusterLinkStub)

	readCreatedClusterLinkResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/regular/read_created_kafka_mirror_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillReturn(
			string(readCreatedClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateTopicStub := wiremock.Post(wiremock.URLPathEqualTo(pauseKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenCreated).
		WillSetStateTo(scenarioStateKafkaMirrorTopicHasBeenPaused).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(updateTopicStub)

	readUpdatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_mirror_topic/regular/read_paused_kafka_mirror_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenPaused).
		WillReturn(
			string(readUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteClusterLinkStub := wiremock.Delete(wiremock.URLPathEqualTo(deleteKafkaMirrorTopicPath)).
		InScenario(kafkaMirrorTopicScenarioName).
		WhenScenarioStateIs(scenarioStateKafkaMirrorTopicHasBeenPaused).
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
				Config: testAccCheckKafkaMirrorTopicConfig(confluentCloudBaseUrl, mockKafkaMirrorTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaMirrorTopicExists(fullKafkaMirrorTopicResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "mirror_topic_name", kafkaMirrorTopicName),
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
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicName)),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "%", numberOfKafkaMirrorTopicResourceAttributes),
				),
			},
			{
				Config: testAccCheckUpdatedKafkaMirrorTopicConfig(confluentCloudBaseUrl, mockKafkaMirrorTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaMirrorTopicExists(fullKafkaMirrorTopicResourceLabel),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "mirror_topic_name", kafkaMirrorTopicName),
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "status", statePaused),
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
					resource.TestCheckResourceAttr(fullKafkaMirrorTopicResourceLabel, "id", fmt.Sprintf("%s/%s/%s", destinationClusterId, clusterLinkName, kafkaMirrorTopicName)),
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

func testAccCheckKafkaMirrorTopicConfig(confluentCloudBaseUrl, mockServerUrl string) string {
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
		kafkaMirrorTopicName, kafkaMirrorTopicName, clusterLinkName,
		destinationClusterId, mockServerUrl, destinationClusterApiKey, destinationClusterApiSecret)
}

func testAccCheckUpdatedKafkaMirrorTopicConfig(confluentCloudBaseUrl, mockServerUrl string) string {
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
	`, confluentCloudBaseUrl, kafkaMirrorTopicResourceLabel, statePaused,
		kafkaMirrorTopicName, kafkaMirrorTopicName, clusterLinkName,
		destinationClusterId, mockServerUrl, destinationClusterApiKey, destinationClusterApiSecret)
}

func testAccCheckKafkaMirrorTopicDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client).kafkaRestClientFactory.CreateKafkaRestClient(mockKafkaMirrorTestServerUrl, destinationClusterId, destinationClusterApiKey, destinationClusterApiSecret, false)
	// Loop through the resources in state, verifying each Cluster Link is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_mirror_topic" {
			continue
		}
		deletedKafkaMirrorTopicId := rs.Primary.ID
		_, response, err := c.apiClient.ClusterLinkingV3Api.ReadKafkaMirrorTopic(c.apiContext(context.Background()), destinationClusterId, clusterLinkName, kafkaMirrorTopicName).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedKafkaMirrorTopicId != "" {
			// Otherwise return the error
			if deletedKafkaMirrorTopicId == rs.Primary.ID {
				return fmt.Errorf("topic (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckKafkaMirrorTopicExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Kafka Mirror Topic has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Kafka Mirror Topic", n)
		}

		return nil
	}
}
