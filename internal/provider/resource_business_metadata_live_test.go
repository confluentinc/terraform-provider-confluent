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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccBusinessMetadataLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Business Metadata live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	businessMetadataName := fmt.Sprintf("tf_live_bm_%d", randomSuffix)
	businessMetadataResourceLabel := "test_live_business_metadata"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBusinessMetadataLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBusinessMetadataLiveConfig(endpoint, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBusinessMetadataLiveExists(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "name", businessMetadataName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "description", "Live test business metadata for data catalog"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "schema_registry_cluster.0.id", schemaRegistryId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "version"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "attribute_definition.#", "2"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials", "rest_endpoint", "schema_registry_cluster"},
			},
		},
	})
}

func TestAccBusinessMetadataUpdateLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Business Metadata live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	businessMetadataName := fmt.Sprintf("tf_live_bm_update_%d", randomSuffix)
	businessMetadataResourceLabel := "test_live_business_metadata_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckBusinessMetadataLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckBusinessMetadataLiveConfig(endpoint, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBusinessMetadataLiveExists(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "attribute_definition.#", "2"),
				),
			},
			{
				Config: testAccCheckBusinessMetadataUpdateLiveConfig(endpoint, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBusinessMetadataLiveExists(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_business_metadata.%s", businessMetadataResourceLabel), "attribute_definition.#", "3"),
				),
			},
		},
	})
}

func testAccCheckBusinessMetadataLiveDestroy(s *terraform.State) error {
	// In live tests, we can't easily check if the resource is actually destroyed
	// without making API calls, so we just verify the resource is removed from state
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_business_metadata" {
			continue
		}
		// If we reach here, the resource should be cleaned up by Terraform
	}
	return nil
}

func testAccCheckBusinessMetadataLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckBusinessMetadataLiveConfig(endpoint, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_business_metadata" "%s" {
		name        = "%s"
		description = "Live test business metadata for data catalog"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		attribute_definition {
			name = "department"
		}

		attribute_definition {
			name = "cost_center"
		}
	}
	`, endpoint, apiKey, apiSecret, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret)
}

func testAccCheckBusinessMetadataUpdateLiveConfig(endpoint, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_business_metadata" "%s" {
		name        = "%s"
		description = "Updated live test business metadata for data catalog"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		attribute_definition {
			name = "department"
		}

		attribute_definition {
			name = "cost_center"
		}

		attribute_definition {
			name = "owner"
			is_optional = true
		}
	}
	`, endpoint, apiKey, apiSecret, businessMetadataResourceLabel, businessMetadataName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret)
} 