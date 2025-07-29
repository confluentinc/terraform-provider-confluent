//go:build live_test && (all || data_catalog)

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

func TestAccTagDataSourceLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Read Schema Registry credentials from environment variables
	schemaRegistryId := os.Getenv("SCHEMA_REGISTRY_ID")
	schemaRegistryApiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	schemaRegistryApiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")
	schemaRegistryRestEndpoint := os.Getenv("SCHEMA_REGISTRY_REST_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if schemaRegistryId == "" || schemaRegistryApiKey == "" || schemaRegistryApiSecret == "" || schemaRegistryRestEndpoint == "" {
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Tag live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	tagName := fmt.Sprintf("tf_live_tag_ds_%d", randomSuffix)
	tagResourceLabel := "test_live_tag_resource"
	tagDataSourceLabel := "test_live_tag_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTagDataSourceLiveConfig(endpoint, tagResourceLabel, tagDataSourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_tag.%s", tagDataSourceLabel), "id",
						fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_tag.%s", tagDataSourceLabel), "name",
						fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "name",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_tag.%s", tagDataSourceLabel), "description",
						fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "description",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_tag.%s", tagDataSourceLabel), "version",
						fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "version",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_tag.%s", tagDataSourceLabel), "entity_types.#",
						fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "entity_types.#",
					),
				),
			},
		},
	})
}

func testAccCheckTagDataSourceLiveConfig(endpoint, tagResourceLabel, tagDataSourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_tag" "%s" {
		name        = "%s"
		description = "Live test tag for data source"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	data "confluent_tag" "%s" {
		name = confluent_tag.%s.name

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, tagDataSourceLabel, tagResourceLabel, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret)
} 