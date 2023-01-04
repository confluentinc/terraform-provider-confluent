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

func TestAccTopicWithEnhancedProviderBlock2(t *testing.T) {
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

	mockTopicTestServerUrl = fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockTopicTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/create_kafka_topic.json")
	createTopicStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTopicHasBeenCreated).
		WillReturn(
			string(createTopicResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createTopicStub)

	readCreatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	updateTopicStub := wiremock.Post(wiremock.URLPathEqualTo(updateKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(updateTopicStub)

	readUpdatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_updated_kafka_topic_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteTopicStub := wiremock.Delete(wiremock.URLPathEqualTo(readKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillSetStateTo(scenarioStateTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteTopicStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTopicDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTopicConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockTopicTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "credentials.0.secret"),
				),
			},
			{
				Config: testAccCheckTopicUpdatedConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockTopicTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "4"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", firstConfigName), firstConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", secondConfigName), secondConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", thirdConfigName), thirdConfigAddedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fourthConfigName), fourthConfigAddedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, "credentials.0.secret"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullTopicResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createTopicStub, fmt.Sprintf("POST %s", createKafkaTopicPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteTopicStub, fmt.Sprintf("DELETE %s", readKafkaTopicPath), expectedCountOne)
}

func testAccCheckTopicConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  kafka_api_key = "%s"
	  kafka_api_secret = "%s"
	  kafka_rest_endpoint = "%s"
      kafka_id = "%s"
	}
	resource "confluent_kafka_topic" "%s" {
	  topic_name = "%s"
	  partitions_count = "%d"
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, clusterId, topicResourceLabel, topicName, partitionCount, firstConfigName, firstConfigValue, secondConfigName, secondConfigValue)
}

func testAccCheckTopicUpdatedConfigWithEnhancedProviderBlock2(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      kafka_api_key = "%s"
      kafka_api_secret = "%s"
      kafka_rest_endpoint = "%s"
      kafka_id = "%s"
    }
	resource "confluent_kafka_topic" "%s" {
	  topic_name = "%s"
	  partitions_count = "%d"
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, clusterId, topicResourceLabel, topicName, partitionCount, firstConfigName, firstConfigValue, secondConfigName, secondConfigUpdatedValue, thirdConfigName, thirdConfigAddedValue, fourthConfigName, fourthConfigAddedValue)
}
