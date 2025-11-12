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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	kafkaClustersDataSourceLabel := "test_live_kafka_clusters_data_source"
	fullKafkaClustersDataSourceLabel := fmt.Sprintf("data.confluent_kafka_clusters.%s", kafkaClustersDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClustersDataSourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Verify we have at least 3 clusters in the environment
					testAccCheckKafkaClustersCountAtLeast(fullKafkaClustersDataSourceLabel, 3),
					// Verify basic structure - just check that clusters exist and have required fields
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.0.id"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.0.display_name"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.0.bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.1.id"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.1.display_name"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.1.bootstrap_endpoint"),
				),
			},
		},
	})
}

func testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClustersDataSourceLabel, apiKey, apiSecret string) string {
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
	`, endpoint, apiKey, apiSecret, kafkaClustersDataSourceLabel)
}

// testAccCheckKafkaClustersCountAtLeast verifies that at least minCount clusters exist
func testAccCheckKafkaClustersCountAtLeast(resourceName string, minCount int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		clustersCountStr, ok := rs.Primary.Attributes["clusters.#"]
		if !ok {
			return fmt.Errorf("clusters count not found in resource attributes")
		}

		clustersCount, err := strconv.Atoi(clustersCountStr)
		if err != nil {
			return fmt.Errorf("error parsing clusters count: %s", err)
		}

		if clustersCount < minCount {
			return fmt.Errorf("expected at least %d clusters, got %d", minCount, clustersCount)
		}

		return nil
	}
}
