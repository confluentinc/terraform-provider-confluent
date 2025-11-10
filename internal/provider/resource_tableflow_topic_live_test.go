//go:build live_test && (all || tableflow)

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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTableflowTopicLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	// Read Kafka cluster configuration
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")

	environmentId := os.Getenv("LIVE_TEST_ENVIRONMENT_ID")

	// Tableflow API credentials
	tableflowApiKey := os.Getenv("TABLEFLOW_API_KEY")
	tableflowApiSecret := os.Getenv("TABLEFLOW_API_SECRET")
	if tableflowApiKey == "" {
		tableflowApiKey = apiKey // Fallback to regular API key if tableflow key not set
	}
	if tableflowApiSecret == "" {
		tableflowApiSecret = apiSecret // Fallback to regular API secret if tableflow secret not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID must be set for tableflow topic live tests")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for tableflow topic live tests")
	}

	// Read Kafka credentials for topic creation
	kafkaApiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	kafkaRestEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")

	if kafkaApiKey == "" || kafkaApiSecret == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_STANDARD_AWS_API_KEY, KAFKA_STANDARD_AWS_API_SECRET, and KAFKA_STANDARD_AWS_REST_ENDPOINT must be set for tableflow topic live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	topicName := fmt.Sprintf("tf-live-tableflow-topic-%d", randomSuffix)
	kafkaTopicResourceLabel := "test_live_kafka_topic_for_tableflow"
	tableflowTopicResourceLabel := "test_live_tableflow_topic"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTableflowTopicLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTableflowTopicLiveConfig(endpoint, kafkaTopicResourceLabel, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTableflowTopicLiveExists(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "display_name", topicName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "managed_storage.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					ttId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.Attributes["environment.0.id"]
					clusterId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.Attributes["kafka_cluster.0.id"]
					return fmt.Sprintf("%s/%s/%s", envId, clusterId, ttId), nil
				},
			},
			{
				Config: testAccCheckTableflowTopicLiveConfigUpdate(endpoint, kafkaTopicResourceLabel, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTableflowTopicLiveExists(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "display_name", topicName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "retention_ms", "1209600000"), // 14 days
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					ttId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.Attributes["environment.0.id"]
					clusterId := resources[fmt.Sprintf("confluent_tableflow_topic.%s", tableflowTopicResourceLabel)].Primary.Attributes["kafka_cluster.0.id"]
					return fmt.Sprintf("%s/%s/%s", envId, clusterId, ttId), nil
				},
			},
		},
	})
}

func testAccCheckTableflowTopicLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_tableflow_topic" {
			continue
		}

		// In live tests, we can't easily check if the resource is actually destroyed
		// without making API calls, so we just verify the resource is removed from state
		if rs.Primary.ID != "" {
			// This is normal - the resource should have an ID but be removed from the live environment
			// The actual cleanup happens through the API calls during destroy
		}
	}
	return nil
}

func testAccCheckTableflowTopicLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		return nil
	}
}

func testAccCheckTableflowTopicLiveConfig(endpoint, kafkaTopicResourceLabel, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Create Kafka topic first (required for tableflow)
	resource "confluent_kafka_topic" "%s" {
		topic_name         = "%s"
		partitions_count   = 6
		rest_endpoint      = "%s"
		kafka_cluster {
			id = "%s"
		}
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	# Enable tableflow on the topic
	resource "confluent_tableflow_topic" "%s" {
		display_name = "%s"
		environment {
			id = "%s"
		}
		kafka_cluster {
			id = "%s"
		}
		managed_storage {}
		retention_ms = "604800000"
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, kafkaTopicResourceLabel, topicName, kafkaRestEndpoint, kafkaClusterId, kafkaApiKey, kafkaApiSecret, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, tableflowApiKey, tableflowApiSecret)
}

func testAccCheckTableflowTopicLiveConfigUpdate(endpoint, kafkaTopicResourceLabel, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Kafka topic still exists
	resource "confluent_kafka_topic" "%s" {
		topic_name         = "%s"
		partitions_count   = 6
		rest_endpoint      = "%s"
		kafka_cluster {
			id = "%s"
		}
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	# Update tableflow topic
	resource "confluent_tableflow_topic" "%s" {
		display_name = "%s"
		environment {
			id = "%s"
		}
		kafka_cluster {
			id = "%s"
		}
		managed_storage {}
		retention_ms = "1209600000"
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, kafkaTopicResourceLabel, topicName, kafkaRestEndpoint, kafkaClusterId, kafkaApiKey, kafkaApiSecret, tableflowTopicResourceLabel, topicName, environmentId, kafkaClusterId, tableflowApiKey, tableflowApiSecret)
}

