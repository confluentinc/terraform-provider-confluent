//go:build live_test && (all || core)

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

func TestAccKsqlClusterDataSourceLive(t *testing.T) {
	t.Skip()
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
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	kafkaApiKey := os.Getenv("KAFKA_STANDARD_AWS_API_KEY")
	kafkaApiSecret := os.Getenv("KAFKA_STANDARD_AWS_API_SECRET")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" || kafkaApiKey == "" || kafkaApiSecret == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID, KAFKA_STANDARD_AWS_API_KEY, and KAFKA_STANDARD_AWS_API_SECRET must be set for ksqlDB Cluster data source live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	ksqlClusterDisplayName := fmt.Sprintf("tf-live-ksql-ds-%d", randomSuffix)
	ksqlClusterResourceLabel := "test_live_ksql_cluster_resource"
	ksqlClusterDataSourceLabel := "test_live_ksql_cluster_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKsqlClusterDataSourceLiveConfig(endpoint, ksqlClusterResourceLabel, ksqlClusterDataSourceLabel, ksqlClusterDisplayName, kafkaClusterId, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the resource was created
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "display_name", ksqlClusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "csu", "4"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "environment.0.id", "env-zyg27z"),

					// Check the data source can find it
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_ksql_cluster.%s", ksqlClusterDataSourceLabel), "id",
						fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_ksql_cluster.%s", ksqlClusterDataSourceLabel), "display_name",
						fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "display_name",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_ksql_cluster.%s", ksqlClusterDataSourceLabel), "csu",
						fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "csu",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_ksql_cluster.%s", ksqlClusterDataSourceLabel), "rest_endpoint",
						fmt.Sprintf("confluent_ksql_cluster.%s", ksqlClusterResourceLabel), "rest_endpoint",
					),
				),
			},
		},
	})
}

func testAccCheckKsqlClusterDataSourceLiveConfig(endpoint, ksqlClusterResourceLabel, ksqlClusterDataSourceLabel, ksqlClusterDisplayName, kafkaClusterId, apiKey, apiSecret, kafkaApiKey, kafkaApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ksql_cluster" "%s" {
		display_name = "%s"
		csu          = 4
		kafka_cluster {
			id = "%s"
		}
		credential_identity {
			id = confluent_service_account.live_ksql_sa.id
		}
		environment {
			id = "env-zyg27z"
		}
		depends_on = [
			confluent_role_binding.live_ksql_sa_rb
		]
	}

	resource "confluent_service_account" "live_ksql_sa" {
		display_name = "%s-sa"
		description  = "Service account for ksqlDB cluster live testing data source"
	}

	resource "confluent_role_binding" "live_ksql_sa_rb" {
		principal   = "User:${confluent_service_account.live_ksql_sa.id}"
		role_name   = "CloudClusterAdmin"
		crn_pattern = "crn://confluent.cloud/organization=424fb7bf-40c2-433f-81a5-c45942a6a539/environment=env-zyg27z/cloud-cluster=%s"
	}

	data "confluent_ksql_cluster" "%s" {
		id = confluent_ksql_cluster.%s.id
		environment {
			id = "env-zyg27z"
		}
	}
	`, endpoint, apiKey, apiSecret, ksqlClusterResourceLabel, ksqlClusterDisplayName, kafkaClusterId, ksqlClusterDisplayName, kafkaClusterId, ksqlClusterDataSourceLabel, ksqlClusterResourceLabel)
}
