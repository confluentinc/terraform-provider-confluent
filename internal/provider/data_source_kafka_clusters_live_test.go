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
)

func TestAccKafkaClustersDataSourceLive(t *testing.T) {
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

	// Read Kafka cluster ID from environment variables
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	kafkaClusterDataSourceLabel := "test_live_kafka_clusters_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClusterDataSourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the data source can find the cluster
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.#", 1),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.display_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.availability"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.cloud"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.region"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_kafka_cluster.%s", kafkaClusterDataSourceLabel), "clusters.0.rest_endpoint"),
				),
			},
		},
	})
}

func testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClusterDataSourceLabel, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	data "confluent_kafka_clusters" "%s" {
		environment {
			id = "env-zyg27z"
		}
	}
	`, endpoint, apiKey, apiSecret, kafkaClusterDataSourceLabel)
}
