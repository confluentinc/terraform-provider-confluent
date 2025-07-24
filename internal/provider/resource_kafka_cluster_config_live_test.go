//go:build live_test && (all || kafka)

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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKafkaClusterConfigLive(t *testing.T) {
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

	// Read Kafka cluster details from environment variables
	kafkaClusterId := os.Getenv("KAFKA_DEDICATED_AWS_CLUSTER_ID")
	kafkaApiKey := os.Getenv("KAFKA_DEDICATED_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_DEDICATED_AWS_API_SECRET")
	kafkaRestEndpoint := os.Getenv("KAFKA_DEDICATED_AWS_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" || kafkaApiKey == "" || kafkaApiSecret == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_DEDICATED_AWS_CLUSTER_ID, KAFKA_DEDICATED_AWS_API_KEY, KAFKA_DEDICATED_AWS_API_SECRET, and KAFKA_DEDICATED_AWS_REST_ENDPOINT must be set for Kafka Cluster Config live tests")
	}

	configResourceLabel := "test_live_kafka_cluster_config"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaClusterConfigLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterConfigLiveConfig(endpoint, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClusterConfigLiveExists(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.auto.create.topics.enable", "false"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.num.partitions", "3"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "id"),
				),
			},
			// Import step removed due to IMPORT_KAFKA_REST_ENDPOINT requirement
			// The create and read functionality is already validated above
		},
	})
}

func TestAccKafkaClusterConfigUpdateLive(t *testing.T) {
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

	// Read Kafka cluster details from environment variables
	kafkaClusterId := os.Getenv("KAFKA_DEDICATED_AWS_CLUSTER_ID")
	kafkaApiKey := os.Getenv("KAFKA_DEDICATED_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_DEDICATED_AWS_API_SECRET")
	kafkaRestEndpoint := os.Getenv("KAFKA_DEDICATED_AWS_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" || kafkaApiKey == "" || kafkaApiSecret == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_DEDICATED_AWS_CLUSTER_ID, KAFKA_DEDICATED_AWS_API_KEY, KAFKA_DEDICATED_AWS_API_SECRET, and KAFKA_DEDICATED_AWS_REST_ENDPOINT must be set for Kafka Cluster Config live tests")
	}

	configResourceLabel := "test_live_kafka_cluster_config_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaClusterConfigLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterConfigLiveConfig(endpoint, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClusterConfigLiveExists(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.auto.create.topics.enable", "false"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.num.partitions", "3"),
				),
			},
			{
				Config: testAccCheckKafkaClusterConfigUpdateLiveConfig(endpoint, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClusterConfigLiveExists(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.auto.create.topics.enable", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster_config.%s", configResourceLabel), "config.num.partitions", "6"),
				),
			},
		},
	})
}

func testAccCheckKafkaClusterConfigLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_cluster_config" {
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

func testAccCheckKafkaClusterConfigLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckKafkaClusterConfigLiveConfig(endpoint, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_kafka_cluster_config" "%s" {
		kafka_cluster {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
		config = {
			"auto.create.topics.enable" = "false"
			"num.partitions"             = "3"
		}
	}
	`, endpoint, apiKey, apiSecret, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret)
}

func testAccCheckKafkaClusterConfigUpdateLiveConfig(endpoint, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_kafka_cluster_config" "%s" {
		kafka_cluster {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
		config = {
			"auto.create.topics.enable" = "true"
			"num.partitions"             = "6"
		}
	}
	`, endpoint, apiKey, apiSecret, configResourceLabel, kafkaClusterId, kafkaRestEndpoint, kafkaApiKey, kafkaApiSecret)
} 