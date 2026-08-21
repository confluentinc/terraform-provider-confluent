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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

// escapeForHCL escapes a raw string value so it can be embedded inside an HCL double-quoted string literal.
// This lets the confluent.*.association configs carry JSON values (containing `"` and `\`) in the test HCL.
func escapeForHCL(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

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

	updatedContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer updatedContainer.Terminate(ctx)

	mockTopicTestServerInitialUrl := initialContainer.URI
	mockTopicTestServerUpdatedUrl := updatedContainer.URI
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
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fifthConfigName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), seventhConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), eighthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "credentials.0.secret", kafkaApiSecret),
				),
			},
			{
				// Step 2: update configs (add segment.bytes, max.compaction.lag.ms; update sixthConfig + both association configs) and delete retention.ms
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
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "6"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", firstConfigName), firstConfigValue),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", secondConfigName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", thirdConfigName), thirdConfigAddedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fourthConfigName), fourthConfigAddedValue),
					resource.TestCheckNoResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", fifthConfigName)),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), seventhConfigUpdatedValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), eighthConfigUpdatedValue),
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
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), seventhConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), eighthConfigValue),
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
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), seventhConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), eighthConfigValue),
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
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.retention.ms", "6789"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), seventhConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), eighthConfigValue),
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
        "%s" = "%s"
        "%s" = "%s"
      }

      credentials {
        key = "%s"
        secret = "%s"
      }
    }
    `, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, secondConfigName, secondConfigValue, sixthConfigName, sixthConfigValue, seventhConfigName, escapeForHCL(seventhConfigValue), eighthConfigName, escapeForHCL(eighthConfigValue), kafkaApiKey, kafkaApiSecret)
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
        "%s" = "%s"
      }

      credentials {
        key = "%s"
        secret = "%s"
      }
    }
    `, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, thirdConfigName, thirdConfigAddedValue, fourthConfigName, fourthConfigAddedValue, sixthConfigName, sixthConfigUpdatedValue, seventhConfigName, escapeForHCL(seventhConfigUpdatedValue), eighthConfigName, escapeForHCL(eighthConfigUpdatedValue), kafkaApiKey, kafkaApiSecret)
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
		"%s" = "%s"
		"%s" = "%s"
	  }

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, partitionCount, mockServerUrl, firstConfigName, firstConfigValue, secondConfigName, secondConfigValue, sixthConfigName, sixthConfigValue, seventhConfigName, escapeForHCL(seventhConfigValue), eighthConfigName, escapeForHCL(eighthConfigValue), kafkaApiKey, kafkaApiSecret)
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

// enrichTopicAssociationConfig simulates how the server-side modification to user's JSON config
func enrichTopicAssociationConfig(t *testing.T, compact, subject string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(compact), &parsed); err != nil {
		t.Fatalf("failed to parse association config %q: %s", compact, err)
	}
	parsed["lifecycle"] = "STRONG"
	parsed["subject"] = subject
	// Pretty-print the embedded schema to mimic the server re-formatting it.
	if schemaValue, ok := parsed["schema"].(string); ok {
		var schemaParsed interface{}
		if err := json.Unmarshal([]byte(schemaValue), &schemaParsed); err == nil {
			if pretty, err := json.MarshalIndent(schemaParsed, "", "  "); err == nil {
				parsed["schema"] = string(pretty)
			}
		}
	}
	enriched, err := json.Marshal(parsed)
	if err != nil {
		t.Fatalf("failed to marshal enriched association config: %s", err)
	}
	return string(enriched)
}

func buildTopicConfigListResponse(t *testing.T, configs [][2]string) string {
	data := make([]interface{}, 0, len(configs))
	for _, kv := range configs {
		data = append(data, map[string]interface{}{
			"kind":         "KafkaTopicConfig",
			"cluster_id":   clusterId,
			"name":         kv[0],
			"value":        kv[1],
			"is_read_only": false,
			"is_sensitive": false,
			"source":       "DYNAMIC_TOPIC_CONFIG",
			"topic_name":   topicName,
			"is_default":   false,
		})
	}
	body, err := json.Marshal(map[string]interface{}{
		"kind": "KafkaTopicConfigList",
		"metadata": map[string]interface{}{
			"self": "https://mock/kafka/v3/clusters/" + clusterId + "/topics/" + topicName + "/configs",
			"next": nil,
		},
		"data": data,
	})
	if err != nil {
		t.Fatalf("failed to build topic config list response: %s", err)
	}
	return string(body)
}

func TestAccTopicAssociationConfigNoDrift(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()
	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/create_kafka_topic.json")
	createTopicStub := wiremock.Post(wiremock.URLPathEqualTo(createKafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTopicHasBeenCreated).
		WillReturn(string(createTopicResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(createTopicStub)

	readCreatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(string(readCreatedTopicResponse), contentTypeJSONHeader, http.StatusOK))

	// The read config stub returns the enriched association values while the HCL writes the compact ones.
	// Tests diff suppression.
	enrichedKeyAssociation := enrichTopicAssociationConfig(t, seventhConfigValue, fmt.Sprintf(":.%s:%s-key", clusterId, topicName))
	enrichedValueAssociation := enrichTopicAssociationConfig(t, eighthConfigValue, fmt.Sprintf(":.%s:%s-value", clusterId, topicName))
	readConfigResponse := buildTopicConfigListResponse(t, [][2]string{
		{firstConfigName, firstConfigValue},
		{secondConfigName, secondConfigValue},
		{sixthConfigName, sixthConfigValue},
		{seventhConfigName, enrichedKeyAssociation},
		{eighthConfigName, enrichedValueAssociation},
	})
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(readConfigResponse, contentTypeJSONHeader, http.StatusOK))
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillReturn(readConfigResponse, contentTypeJSONHeader, http.StatusOK))

	deleteTopicStub := wiremock.Delete(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenCreated).
		WillSetStateTo(scenarioStateTopicHasBeenDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent)
	_ = wiremockClient.StubFor(deleteTopicStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicScenarioName).
		WhenScenarioStateIs(scenarioStateTopicHasBeenDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNotFound))

	// Verifies the value stored in state is semantically equal to the user input.
	assertAssociationEquivalent := func(compact string) resource.CheckResourceAttrWithFunc {
		return func(value string) error {
			if !associationConfigsEquivalent(value, compact) {
				return fmt.Errorf("state value %q is not semantically equivalent to %q", value, compact)
			}
			return nil
		}
	}

	// Assert that the server adds "lifecycle"/"subject" fields in addition to user config.
	assertStateIsEnriched := func(value string) error {
		for _, field := range []string{`"lifecycle"`, `"subject"`} {
			if !strings.Contains(value, field) {
				return fmt.Errorf("expected server-enriched state to contain %s, got %q", field, value)
			}
		}
		return nil
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckTopicDestroy(s, mockServerUrl)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTopicConfig(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicResourceLabel),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, "config.%", "5"),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", firstConfigName), firstConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", secondConfigName), secondConfigValue),
					resource.TestCheckResourceAttr(fullTopicResourceLabel, fmt.Sprintf("config.%s", sixthConfigName), sixthConfigValue),
					resource.TestCheckResourceAttrWith(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), assertAssociationEquivalent(seventhConfigValue)),
					resource.TestCheckResourceAttrWith(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), assertAssociationEquivalent(eighthConfigValue)),
					resource.TestCheckResourceAttrWith(fullTopicResourceLabel, fmt.Sprintf("config.%s", seventhConfigName), assertStateIsEnriched),
					resource.TestCheckResourceAttrWith(fullTopicResourceLabel, fmt.Sprintf("config.%s", eighthConfigName), assertStateIsEnriched),
				),
			},
			{
				// Re-plan with the same config
				Config:   testAccCheckTopicConfig(confluentCloudBaseUrl, mockServerUrl),
				PlanOnly: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createTopicStub, fmt.Sprintf("POST %s", createKafkaTopicPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteTopicStub, fmt.Sprintf("DELETE %s", kafkaTopicPath), expectedCountOne)
}

// The following unit tests cover the semantic diff-suppression for SR Association (Project Odyssey) configs.

// userAssociation is what the user writes via jsonencode(...): a compact JSON document whose "schema" is a JSON string.
const userAssociation = `{"schema":"{\"name\":\"TestKeyRecord\",\"type\":\"record\"}","schemaType":"AVRO"}`

// enrichedServerAssociation is what the server stores and returns。
const enrichedServerAssociation = `{"lifecycle":"STRONG","schema":"{\n  \"type\": \"record\",\n  \"name\": \"TestKeyRecord\"\n}","schemaType":"AVRO","subject":":.lkc-abc123:topic-key"}`

func TestParseAssociationConfig(t *testing.T) {
	tests := []struct {
		name  string
		input string
		ok    bool
	}{
		{name: "valid object parses", input: userAssociation, ok: true},
		{name: "empty value does not parse", input: "", ok: false},
		{name: "non-object json does not parse", input: `"just-a-string"`, ok: false},
		{name: "invalid json does not parse", input: "not-json", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := parseAssociationConfig(tt.input)
			if ok != tt.ok {
				t.Errorf("parseAssociationConfig(%q) ok = %v, want %v", tt.input, ok, tt.ok)
			}
		})
	}
}

func TestNormalizeAssociationValue(t *testing.T) {
	tests := []struct {
		name string
		a    interface{}
		b    interface{}
		want bool
	}{
		{
			name: "embedded schema ignores whitespace/formatting",
			a:    "{\n  \"type\": \"record\",\n  \"name\": \"R\"\n}",
			b:    `{"name":"R","type":"record"}`,
			want: true,
		},
		{
			name: "embedded schema with different content differs",
			a:    `{"name":"R","type":"record"}`,
			b:    `{"name":"R2","type":"record"}`,
			want: false,
		},
		{
			name: "plain string values compared verbatim (equal)",
			a:    "AVRO",
			b:    "AVRO",
			want: true,
		},
		{
			name: "plain string values compared verbatim (different)",
			a:    "STRONG",
			b:    "WEAK",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAssociationValue(tt.a) == normalizeAssociationValue(tt.b); got != tt.want {
				t.Errorf("normalizeAssociationValue(%v)==normalizeAssociationValue(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestAssociationConfigsEquivalent(t *testing.T) {
	tests := []struct {
		name string
		old  string
		new  string
		want bool
	}{
		{
			name: "server-added fields the user omitted are ignored",
			old:  enrichedServerAssociation,
			new:  userAssociation,
			want: true,
		},
		{
			name: "identical values are equivalent",
			old:  userAssociation,
			new:  userAssociation,
			want: true,
		},
		{
			// The user explicitly set subject to the same value the server stored -> equivalent.
			name: "user-set field matching the server is equivalent",
			old:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","subject":"my-subject"}`,
			new:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","subject":"my-subject"}`,
			want: true,
		},
		{
			// The user explicitly set subject and changed it -> real change, not suppressed.
			name: "user-set field changed by the user is not suppressed",
			old:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","subject":"old-subject"}`,
			new:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","subject":"new-subject"}`,
			want: false,
		},
		{
			// The user set lifecycle explicitly and changed it -> real change, not suppressed.
			name: "user-set lifecycle changed by the user is not suppressed",
			old:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","lifecycle":"WEAK"}`,
			new:  `{"schema":"{\"name\":\"R\",\"type\":\"record\"}","schemaType":"AVRO","lifecycle":"STRONG"}`,
			want: false,
		},
		{
			name: "different embedded schema content is not suppressed",
			old:  enrichedServerAssociation,
			new:  `{"schema":"{\"name\":\"RenamedRecord\",\"type\":\"record\"}","schemaType":"AVRO"}`,
			want: false,
		},
		{
			name: "different schemaType is not suppressed",
			old:  enrichedServerAssociation,
			new:  `{"schema":"{\"name\":\"TestKeyRecord\",\"type\":\"record\"}","schemaType":"PROTOBUF"}`,
			want: false,
		},
		{
			name: "user adding a field inside the schema is not suppressed",
			old:  `{"lifecycle":"STRONG","schema":"{\"name\":\"R\",\"type\":\"record\",\"fields\":[{\"name\":\"f1\",\"type\":\"string\"}]}","schemaType":"AVRO","subject":"s"}`,
			new:  `{"schema":"{\"name\":\"R\",\"type\":\"record\",\"fields\":[{\"name\":\"f1\",\"type\":\"string\"},{\"name\":\"f2\",\"type\":\"int\"}]}","schemaType":"AVRO"}`,
			want: false,
		},
		{
			name: "non-json values fall back to exact comparison (equal)",
			old:  "plain-value",
			new:  "plain-value",
			want: true,
		},
		{
			name: "non-json values fall back to exact comparison (different)",
			old:  "plain-value",
			new:  "other-value",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := associationConfigsEquivalent(tt.old, tt.new); got != tt.want {
				t.Errorf("associationConfigsEquivalent(%q, %q) = %v, want %v", tt.old, tt.new, got, tt.want)
			}
		})
	}
}

func TestSuppressTopicConfigDiff(t *testing.T) {
	tests := []struct {
		name string
		key  string
		old  string
		new  string
		want bool
	}{
		{
			name: "association config with server enrichment is suppressed",
			key:  fmt.Sprintf("%s.confluent.key.association", paramConfigs),
			old:  enrichedServerAssociation,
			new:  userAssociation,
			want: true,
		},
		{
			name: "association config with a real change is not suppressed",
			key:  fmt.Sprintf("%s.confluent.value.association", paramConfigs),
			old:  enrichedServerAssociation,
			new:  `{"schema":"{\"name\":\"RenamedRecord\",\"type\":\"record\"}","schemaType":"AVRO"}`,
			want: false,
		},
		{
			name: "non-association config falls back to exact comparison (equal)",
			key:  fmt.Sprintf("%s.cleanup.policy", paramConfigs),
			old:  "compact",
			new:  "compact",
			want: false, // returns false so the SDK applies its default (exact) comparison
		},
		{
			name: "non-association config falls back to exact comparison (different)",
			key:  fmt.Sprintf("%s.cleanup.policy", paramConfigs),
			old:  "compact",
			new:  "delete",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suppressTopicConfigDiff(tt.key, tt.old, tt.new, nil); got != tt.want {
				t.Errorf("suppressTopicConfigDiff(%q, %q, %q) = %v, want %v", tt.key, tt.old, tt.new, got, tt.want)
			}
		})
	}
}
