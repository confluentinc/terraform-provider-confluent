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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	topicDataSourceScenarioName = "confluent_kafka_topic Data Source Lifecycle"
)

var fullTopicDataSourceLabel = fmt.Sprintf("data.confluent_kafka_topic.%s", topicResourceLabel)

func TestAccDataSourceTopic(t *testing.T) {
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

	readCreatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kafkaTopicPath)).
		InScenario(topicDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedTopicResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedTopicConfigResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicConfigPath)).
		InScenario(topicDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedTopicConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceTopicConfig(confluentCloudBaseUrl, mockTopicTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTopicExists(fullTopicDataSourceLabel),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "kafka_cluster.0.id", clusterId),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "id", fmt.Sprintf("%s/%s", clusterId, topicName)),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "%", numberOfResourceAttributes),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "topic_name", topicName),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "partitions_count", strconv.Itoa(partitionCount)),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "rest_endpoint", mockTopicTestServerUrl),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.%", "3"),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.retention.ms", "6789"),
				),
			},
		},
	})
}

func testAccCheckDataSourceTopicConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_kafka_topic" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	
	  topic_name = "%s"
	  rest_endpoint = "%s"

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, mockServerUrl, kafkaApiKey, kafkaApiSecret)
}
