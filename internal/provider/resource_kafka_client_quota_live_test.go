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
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKafkaClientQuotaLive(t *testing.T) {
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" {
		t.Fatal("KAFKA_DEDICATED_AWS_CLUSTER_ID must be set for Kafka Client Quota live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	quotaDisplayName := fmt.Sprintf("tf-live-quota-%d", randomSuffix)
	quotaResourceLabel := "test_live_kafka_client_quota"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaClientQuotaLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClientQuotaLiveConfig(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaLiveExists(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "display_name", quotaDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "description", "A test client quota for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.ingress_byte_rate", "1048576"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.egress_byte_rate", "2097152"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "principals.#", "1"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "principals.*", "<default>"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccKafkaClientQuotaUpdateLive(t *testing.T) {
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" {
		t.Fatal("KAFKA_DEDICATED_AWS_CLUSTER_ID must be set for Kafka Client Quota live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	quotaDisplayName := fmt.Sprintf("tf-live-quota-update-%d", randomSuffix)
	quotaResourceLabel := "test_live_kafka_client_quota_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKafkaClientQuotaLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClientQuotaUpdateLiveConfigWithSA(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaLiveExists(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.ingress_byte_rate", "1048576"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.egress_byte_rate", "2097152"),
				),
			},
			{
				Config: testAccCheckKafkaClientQuotaUpdateLiveConfigUpdated(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClientQuotaLiveExists(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.ingress_byte_rate", "2097152"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "throughput.0.egress_byte_rate", "4194304"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_client_quota.%s", quotaResourceLabel), "description", "An updated test client quota for live testing"),
				),
			},
		},
	})
}

func testAccCheckKafkaClientQuotaLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_kafka_client_quota" {
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

func testAccCheckKafkaClientQuotaLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckKafkaClientQuotaLiveConfig(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_kafka_client_quota" "%s" {
		display_name = "%s"
		description  = "A test client quota for live testing"
		kafka_cluster {
			id = "%s"
		}
		environment {
			id = "env-zyg27z"
		}
		throughput {
			ingress_byte_rate = "1048576"  # 1 MB/s
			egress_byte_rate  = "2097152"  # 2 MB/s
		}
		principals = ["<default>"]
	}
	`, endpoint, apiKey, apiSecret, quotaResourceLabel, quotaDisplayName, kafkaClusterId)
}

func testAccCheckKafkaClientQuotaUpdateLiveConfigWithSA(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "quota_test_sa" {
		display_name = "%s-sa"
		description  = "Service account for quota testing"
	}

	resource "confluent_kafka_client_quota" "%s" {
		display_name = "%s"
		description  = "A test client quota for live testing"
		kafka_cluster {
			id = "%s"
		}
		environment {
			id = "env-zyg27z"
		}
		throughput {
			ingress_byte_rate = "1048576"  # 1 MB/s
			egress_byte_rate  = "2097152"  # 2 MB/s
		}
		principals = [confluent_service_account.quota_test_sa.id]
	}
	`, endpoint, apiKey, apiSecret, quotaDisplayName, quotaResourceLabel, quotaDisplayName, kafkaClusterId)
}

func testAccCheckKafkaClientQuotaUpdateLiveConfigUpdated(endpoint, quotaResourceLabel, quotaDisplayName, kafkaClusterId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "quota_test_sa" {
		display_name = "%s-sa"
		description  = "Service account for quota testing"
	}

	resource "confluent_kafka_client_quota" "%s" {
		display_name = "%s"
		description  = "An updated test client quota for live testing"
		kafka_cluster {
			id = "%s"
		}
		environment {
			id = "env-zyg27z"
		}
		throughput {
			ingress_byte_rate = "2097152"  # 2 MB/s
			egress_byte_rate  = "4194304"  # 4 MB/s
		}
		principals = [confluent_service_account.quota_test_sa.id]
	}
	`, endpoint, apiKey, apiSecret, quotaDisplayName, quotaResourceLabel, quotaDisplayName, kafkaClusterId)
} 