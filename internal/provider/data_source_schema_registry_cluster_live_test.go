//go:build live_test && (all || schema_registry)

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

func TestAccSchemaRegistryClusterDataSourceLive(t *testing.T) {
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

	// Read Schema Registry cluster ID from environment variables
	schemaRegistryId := os.Getenv("SCHEMA_REGISTRY_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if schemaRegistryId == "" {
		t.Fatal("SCHEMA_REGISTRY_ID must be set for Schema Registry Cluster data source live tests")
	}

	schemaRegistryDataSourceLabel := "test_live_schema_registry_cluster_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterDataSourceLiveConfig(endpoint, schemaRegistryDataSourceLabel, schemaRegistryId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the data source can find the cluster
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "id", schemaRegistryId),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "display_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "cloud"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "region"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "package"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryDataSourceLabel), "rest_endpoint"),
				),
			},
		},
	})
}

func testAccCheckSchemaRegistryClusterDataSourceLiveConfig(endpoint, schemaRegistryDataSourceLabel, schemaRegistryId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	data "confluent_schema_registry_cluster" "%s" {
		id = "%s"
		environment {
			id = "env-zyg27z"
		}
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryDataSourceLabel, schemaRegistryId)
} 