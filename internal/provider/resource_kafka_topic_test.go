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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateTopicHasBeenCreated       = "A new topic has been just created"
	scenarioStateTopicHasBeenUpdated       = "A new topic has been just updated"
	scenarioStateTopicHasBeenDeleted       = "The topic has been deleted"
	scenarioStateTopicHasBeenUpdateCreated = "The topic has been update created"
	scenarioStateTopicHasBeenDeletedUpdate = "The topic has been update deleted"
	topicScenarioName                      = "confluent_kafka_topic Resource Lifecycle"
	clusterId                              = "lkc-190073"
	partitionCount                         = 4
	partitionCountUpdated                  = 6
	partitionCountUpdated2                 = 2
	firstConfigName                        = "max.message.bytes"
	firstConfigValue                       = "12345"
	secondConfigName                       = "retention.ms"
	secondConfigValue                      = "6789"
	secondConfigUpdatedValue               = "67890"
	thirdConfigName                        = "segment.bytes"
	thirdConfigAddedValue                  = "104857600"
	fourthConfigName                       = "max.compaction.lag.ms"
	fifthConfigName                        = "confluent.topic.type"
	sixthConfigName                        = "confluent.schema.validation.context.name"
	sixthConfigValue                       = "default"
	sixthConfigUpdatedValue                = ".mycontext"
	fourthConfigAddedValue                 = "604800000"
	topicName                              = "test_topic_name"
	topicResourceLabel                     = "test_topic_resource_label"
	kafkaApiKey                            = "test_key"
	kafkaApiSecret                         = "test_secret"
	numberOfResourceAttributes             = "7"
)

var fullTopicResourceLabel = fmt.Sprintf("confluent_kafka_topic.%s", topicResourceLabel)
var createKafkaTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics", clusterId)
var kafkaTopicPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics/%s", clusterId, topicName)
var readKafkaTopicConfigPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics/%s/configs", clusterId, topicName)
var updateKafkaTopicConfigPath = fmt.Sprintf("/kafka/v3/clusters/%s/topics/%s/configs:alter", clusterId, topicName)

func TestAccTopic(t *testing.T) {
	ctx := context.Background()

	initialContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer initialContainer.Terminate(ctx)

	updatedServerUrl, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer updatedServerUrl.Terminate(ctx)

	mockTopicTestServerInitialUrl := initialContainer.URI
	mockTopicTestServerUpdatedUrl := updatedServerUrl.URI
	confluentCloudBaseUrl := ""
	initialClient := wiremock.NewClient(mockTopicTestServerInitialUrl)
	updatedClient := wiremock.NewClient(mockTopicTestServerUpdatedUrl)
	// nolint:errcheck
	defer initialClient.Reset()
	defer updatedClient.Reset()

	// nolint:errcheck
	defer initialClient.ResetAllScenarios()
	defer updatedClient.ResetAllScenarios()

	// WireMock doesn't support scenario state transitions between different client instances.
	// Each WireMock container maintains its own independent scenario state, so when we switch
	// from initialClient (port 8080) to updatedClient (port 8081) between test steps,
	// the scenario state doesn't carry over. This hack creates a dummy endpoint that transitions
	// the state from "Started" to "A new topic has been just created" on the second instance.
	dummyPath := "/state-sync"
	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dummyPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTopicHasBeenCreated).
		WillReturn("OK", contentTypeJSONHeader, http.StatusOK))

	// Trigger the state transition by calling the dummy endpoint
	http.Get(mockTopicTestServerUpdatedUrl + dummyPath)

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
	_ = initialClient.StubFor(createTopicStub)

	readCreatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic.json")
	_ = initialClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = initialClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic_config.json")
	_ = initialClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = initialClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
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
	_ = updatedClient.StubFor(updateTopicStub)

	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_updated_kafka_topic_config.json")
	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteTopicStub := wiremock.Delete(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillSetStateTo(scenarioStateTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = updatedClient.StubFor(deleteTopicStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_KAFKA_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_KAFKA_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_KAFKA_REST_ENDPOINT", mockTopicTestServerUpdatedUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_KAFKA_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckTopicDestroy(s, mockTopicTestServerUpdatedUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTopicConfig(confluentCloudBaseUrl, mockTopicTestServerInitialUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "rest_endpoint", mockTopicTestServerInitialUrl),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fifthConfigName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				Config: testAccCheckTopicUpdatedConfig(confluentCloudBaseUrl, mockTopicTestServerUpdatedUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "rest_endpoint", mockTopicTestServerUpdatedUrl),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", firstConfigName), firstConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", secondConfigName), secondConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", thirdConfigName), thirdConfigAddedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fourthConfigName), fourthConfigAddedValue),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fifthConfigName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
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

	checkStubCount(t, initialClient, createTopicStub, fmt.Sprintf("POST %s", createKafkaTopicPath), expectedCountOne)
	checkStubCount(t, updatedClient, deleteTopicStub, fmt.Sprintf("DELETE %s", kafkaTopicPath), expectedCountOne)
}

func TestAccTopicPartition(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockTopicTestServerUrl := wiremockContainer.URI
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
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
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

	updateTopicStub := wiremock.Patch(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(updateTopicStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_updated_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillReturn(
			string(readUpdatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteTopicStubUpdate := wiremock.Delete(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdated).
		WillSetStateTo(scenarioStateTopicHasBeenDeletedUpdate).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteTopicStubUpdate)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenDeletedUpdate).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	createTopicUpdateResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/create_kafka_topic.json")
	createTopicUpdateStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenDeletedUpdate).
		WillSetStateTo(scenarioStateTopicHasBeenUpdateCreated).
		WillReturn(
			string(createTopicUpdateResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createTopicUpdateStub)

	readCreatedTopicUpdateResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_create_updated_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdateCreated).
		WillReturn(
			string(readCreatedTopicUpdateResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedUpdatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdateCreated).
		WillReturn(
			string(readCreatedUpdatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteTopicStub := wiremock.Delete(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenUpdateCreated).
		WillSetStateTo(scenarioStateTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteTopicStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_KAFKA_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_KAFKA_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_KAFKA_REST_ENDPOINT", mockTopicTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_KAFKA_API_KEY")
		_ = os.Unsetenv("IMPORT_KAFKA_API_SECRET")
		_ = os.Unsetenv("IMPORT_KAFKA_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckTopicDestroy(s, mockTopicTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTopicConfig(confluentCloudBaseUrl, mockTopicTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "rest_endpoint", mockTopicTestServerUrl),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				Config: testAccCheckTopicPartition(confluentCloudBaseUrl, mockTopicTestServerUrl, partitionCountUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCountUpdated)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "rest_endpoint", mockTopicTestServerUrl),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				Config: testAccCheckTopicPartition(confluentCloudBaseUrl, mockTopicTestServerUrl, partitionCountUpdated2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "partitions_count", strconv.Itoa(partitionCountUpdated2)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "rest_endpoint", mockTopicTestServerUrl),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
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

	checkStubCount(t, wiremockClient, createTopicStub, fmt.Sprintf("POST %s", createKafkaTopicPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteTopicStub, fmt.Sprintf("DELETE %s", kafkaTopicPath), expectedCountTwo)
}

func testAccCheckTopicDestroy(s *terraform.State, url string) error {
	testClient := testAccProvider.Meta().(*Client)
	c := testClient.kafkaRestClientFactory.CreateKafkaRestClient(url, clusterId, kafkaApiKey, kafkaApiSecret, false, false, testClient.oauthToken)
	// Loop through the resources in state, verifying each Kafka topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_topic" {
			continue
		}
		deletedTopicId := rs.Primary.ID
		_, response, err := c.apiClient.TopicV3Api.GetKafkaTopic(c.apiContext(context.Background()), clusterId, topicName).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedTopicId != "" {
			// Otherwise return the error
			if deletedTopicId == rs.Primary.ID {
				return fmt.Errorf("topic (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckTopicConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
      endpoint = "%s"
    }
    resource "confluent_kafka_topic" "%s" {
      kafka_cluster {
        id = "%s"
      }
    
      topic_name = "%s"
      partitions_count = "%d"
      rest_endpoint = "%s"
    
      config = {
        "%s" = "%s"
        "%s" = "%s"
        "%s" = "%s"
      }

      credentials {
        key = "%s"
        secret = "%s"
      }
    }
    `, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, secondConfigName, secondConfigValue, sixthConfigName, sixthConfigValue, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckTopicUpdatedConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
      endpoint = "%s"
    }
    resource "confluent_kafka_topic" "%s" {
      kafka_cluster {
        id = "%s"
      }
    
      topic_name = "%s"
      partitions_count = "%d"
      rest_endpoint = "%s"
    
      config = {
        "%s" = "%s"
        "%s" = "%s"
        "%s" = "%s"
        "%s" = "%s"
        "%s" = "%s"
      }

      credentials {
        key = "%s"
        secret = "%s"
      }
    }
    `, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, secondConfigName, secondConfigUpdatedValue, thirdConfigName, thirdConfigAddedValue, fourthConfigName, fourthConfigAddedValue, sixthConfigName, sixthConfigUpdatedValue, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckTopicPartition(confluentCloudBaseUrl, mockServerUrl string, partitionCount int) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	resource "confluent_kafka_topic" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	
	  topic_name = "%s"
	  partitions_count = "%d"
	  rest_endpoint = "%s"
	
	  config = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, secondConfigName, secondConfigValue, sixthConfigName, sixthConfigValue, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckTopicExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s topic has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s topic", n)
		}

		return nil
	}
}
