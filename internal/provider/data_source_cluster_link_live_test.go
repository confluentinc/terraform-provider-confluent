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
)

func TestAccClusterLinkDataSourceLive(t *testing.T) {
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

	// Read Standard Kafka cluster details (source)
	standardClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	standardApiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	standardApiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")
	standardRestEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")

	// Read Dedicated Kafka cluster details (destination)
	dedicatedClusterId := os.Getenv("KAFKA_DEDICATED_AWS_CLUSTER_ID")
	dedicatedApiKey := os.Getenv("KAFKA_DEDICATED_AWS_API_KEY")
	dedicatedApiSecret := os.Getenv("KAFKA_DEDICATED_AWS_API_SECRET")
	dedicatedRestEndpoint := os.Getenv("KAFKA_DEDICATED_AWS_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if standardClusterId == "" || standardApiKey == "" || standardApiSecret == "" || standardRestEndpoint == "" {
		t.Fatal("KAFKA_STANDARD_AWS_* environment variables must be set for Cluster Link data source live tests (source cluster)")
	}

	if dedicatedClusterId == "" || dedicatedApiKey == "" || dedicatedApiSecret == "" || dedicatedRestEndpoint == "" {
		t.Fatal("KAFKA_DEDICATED_AWS_* environment variables must be set for Cluster Link data source live tests (destination cluster)")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	linkName := fmt.Sprintf("tf-live-cluster-link-ds-%d", randomSuffix)
	clusterLinkResourceLabel := "test_live_cluster_link_resource"
	clusterLinkDataSourceLabel := "test_live_cluster_link_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterLinkDataSourceLiveConfig(endpoint, clusterLinkResourceLabel, clusterLinkDataSourceLabel, linkName, standardClusterId, dedicatedClusterId, dedicatedRestEndpoint, apiKey, apiSecret, standardApiKey, standardApiSecret, dedicatedApiKey, dedicatedApiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the resource attributes
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel), "link_name", linkName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel), "link_mode", "DESTINATION"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel), "connection_mode", "OUTBOUND"),
					
					// Check the data source attributes
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "link_name", linkName),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "kafka_cluster.0.id", dedicatedClusterId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "cluster_link_id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "link_state"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "id"),
					
					// Ensure data source and resource have matching attributes
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel), "cluster_link_id", fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "cluster_link_id"),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_cluster_link.%s", clusterLinkResourceLabel), "link_name", fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel), "link_name"),
				),
			},
		},
	})
}

func testAccCheckClusterLinkDataSourceLiveConfig(endpoint, clusterLinkResourceLabel, clusterLinkDataSourceLabel, linkName, standardClusterId, dedicatedClusterId, dedicatedRestEndpoint, apiKey, apiSecret, standardApiKey, standardApiSecret, dedicatedApiKey, dedicatedApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Get the Standard cluster details to access bootstrap_endpoint
	data "confluent_kafka_cluster" "standard_cluster" {
		id = "%s"
		environment {
			id = "env-zyg27z"
		}
	}

	# Create a cluster link from Standard (source) to Dedicated (destination)
	resource "confluent_cluster_link" "%s" {
		link_name = "%s"
		source_kafka_cluster {
			id = "%s"
			bootstrap_endpoint = data.confluent_kafka_cluster.standard_cluster.bootstrap_endpoint
			credentials {
				key    = "%s"
				secret = "%s"
			}
		}
		destination_kafka_cluster {
			id = "%s"
			rest_endpoint = "%s"
			credentials {
				key    = "%s"
				secret = "%s"
			}
		}
	}

	# Read the cluster link using data source
	data "confluent_cluster_link" "%s" {
		link_name = confluent_cluster_link.%s.link_name
		kafka_cluster {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, standardClusterId, clusterLinkResourceLabel, linkName, standardClusterId, standardApiKey, standardApiSecret, dedicatedClusterId, dedicatedRestEndpoint, dedicatedApiKey, dedicatedApiSecret, clusterLinkDataSourceLabel, clusterLinkResourceLabel, dedicatedClusterId, dedicatedRestEndpoint, dedicatedApiKey, dedicatedApiSecret)
} 