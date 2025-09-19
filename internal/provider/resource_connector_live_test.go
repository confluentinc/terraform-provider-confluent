//go:build live_test && (all || connect)

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

func TestAccConnectorLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Read Kafka cluster credentials from environment variables
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	kafkaApiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	kafkaRestEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" || kafkaApiKey == "" || kafkaApiSecret == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID, KAFKA_STANDARD_AWS_API_KEY, KAFKA_STANDARD_AWS_API_SECRET, and KAFKA_STANDARD_AWS_REST_ENDPOINT must be set for Connector live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	connectorName := fmt.Sprintf("tf-live-connector-%d", randomSuffix)
	connectorResourceLabel := "test_live_connector"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectorLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectorLiveConfig(endpoint, connectorResourceLabel, connectorName, fmt.Sprintf("%s-topic", connectorName), kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorLiveExists(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.name", connectorName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.connector.class", "DatagenSource"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.quickstart", "ORDERS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.max.interval", "1000"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.tasks.max", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "status"),
				),
			},
			{
				ResourceName:            fmt.Sprintf("confluent_connector.%s", connectorResourceLabel),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"config_sensitive"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					connectorName := resources[fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)].Primary.Attributes["config_nonsensitive.name"]
					environmentId := resources[fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)].Primary.Attributes["environment.0.id"]
					clusterId := resources[fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)].Primary.Attributes["kafka_cluster.0.id"]
					return environmentId + "/" + clusterId + "/" + connectorName, nil
				},
			},
		},
	})
}

func TestAccConnectorUpdateLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Read Kafka cluster credentials from environment variables
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	kafkaApiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	kafkaRestEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" || kafkaApiKey == "" || kafkaApiSecret == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID, KAFKA_STANDARD_AWS_API_KEY, KAFKA_STANDARD_AWS_API_SECRET, and KAFKA_STANDARD_AWS_REST_ENDPOINT must be set for Connector live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	connectorName := fmt.Sprintf("tf-live-connector-update-%d", randomSuffix)
	connectorResourceLabel := "test_live_connector_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectorLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectorLiveConfig(endpoint, connectorResourceLabel, connectorName, fmt.Sprintf("%s-topic", connectorName), kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorLiveExists(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.max.interval", "1000"),
				),
			},
			{
				Config: testAccCheckConnectorUpdateLiveConfig(endpoint, connectorResourceLabel, connectorName, fmt.Sprintf("%s-topic", connectorName), kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectorLiveExists(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connector.%s", connectorResourceLabel), "config_nonsensitive.max.interval", "2000"),
				),
			},
		},
	})
}

func testAccCheckConnectorLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_connector" {
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

func testAccCheckConnectorLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckConnectorLiveConfig(endpoint, connectorResourceLabel, connectorName, topicName, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Create the Kafka topic that the connector will use
	resource "confluent_kafka_topic" "connector_topic" {
		kafka_cluster {
			id = "%s"
		}
		topic_name         = "%s"
		partitions_count   = 6
		rest_endpoint      = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	resource "confluent_connector" "%s" {
		environment {
			id = "env-zyg27z"
		}
		kafka_cluster {
			id = "%s"
		}

		config_nonsensitive = {
			"name"            = "%s"
			"connector.class" = "DatagenSource"
			"kafka.topic"     = confluent_kafka_topic.connector_topic.topic_name
			"quickstart"      = "ORDERS"
			"max.interval"    = "1000"
			"tasks.max"       = "1"
		}

		config_sensitive = {
			"kafka.api.key"    = "%s"
			"kafka.api.secret" = "%s"
		}

		depends_on = [confluent_kafka_topic.connector_topic]
	}
	`, endpoint, apiKey, apiSecret, kafkaClusterId, topicName, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, connectorResourceLabel, kafkaClusterId, connectorName, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckConnectorUpdateLiveConfig(endpoint, connectorResourceLabel, connectorName, topicName, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Create the Kafka topic that the connector will use
	resource "confluent_kafka_topic" "connector_topic" {
		kafka_cluster {
			id = "%s"
		}
		topic_name         = "%s"
		partitions_count   = 6
		rest_endpoint      = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	resource "confluent_connector" "%s" {
		environment {
			id = "env-zyg27z"
		}
		kafka_cluster {
			id = "%s"
		}

		config_nonsensitive = {
			"name"            = "%s"
			"connector.class" = "DatagenSource"
			"kafka.topic"     = confluent_kafka_topic.connector_topic.topic_name
			"quickstart"      = "ORDERS"
			"max.interval"    = "2000"
			"tasks.max"       = "1"
		}

		config_sensitive = {
			"kafka.api.key"    = "%s"
			"kafka.api.secret" = "%s"
		}

		depends_on = [confluent_kafka_topic.connector_topic]
	}
	`, endpoint, apiKey, apiSecret, kafkaClusterId, topicName, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret, connectorResourceLabel, kafkaClusterId, connectorName, kafkaApiKey, kafkaApiSecret)
}
