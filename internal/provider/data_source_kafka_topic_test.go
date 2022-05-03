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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	topicDataSourceScenarioName = "confluentcloud_kafka_topic Data Source Lifecycle"
)

var fullTopicDataSourceLabel = fmt.Sprintf("data.confluentcloud_kafka_topic.%s", topicResourceLabel)

func TestAccDataSourceTopic(t *testing.T) {
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

	readCreatedTopicResponse, _ := ioutil.ReadFile("../testdata/kafka_topic/read_created_kafka_topic.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKafkaTopicPath)).
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

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("KAFKA_API_KEY", kafkaApiKey)
	_ = os.Setenv("KAFKA_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("KAFKA_HTTP_ENDPOINT", mockTopicTestServerUrl)
	defer func() {
		_ = os.Unsetenv("KAFKA_API_KEY")
		_ = os.Unsetenv("KAFKA_API_SECRET")
		_ = os.Unsetenv("KAFKA_HTTP_ENDPOINT")
	}()

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
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "http_endpoint", mockTopicTestServerUrl),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.%", "2"),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.max.message.bytes", "12345"),
					resource.TestCheckResourceAttr(fullTopicDataSourceLabel, "config.retention.ms", "6789"),
				),
			},
		},
	})
}

func testAccCheckDataSourceTopicConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluentcloud" {
      endpoint = "%s"
    }
	data "confluentcloud_kafka_topic" "%s" {
	  kafka_cluster {
        id = "%s"
      }
	
	  topic_name = "%s"
	  http_endpoint = "%s"

	  credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, topicResourceLabel, clusterId, topicName, mockServerUrl, kafkaApiKey, kafkaApiSecret)
}
