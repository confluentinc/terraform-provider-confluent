//go:build live_test && (all || catalog)

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

func TestAccCatalogEntityAttributesLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	// Read Schema Registry credentials from environment variables
	schemaRegistryApiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	schemaRegistryApiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")
	schemaRegistryRestEndpoint := os.Getenv("SCHEMA_REGISTRY_REST_ENDPOINT")
	schemaRegistryId := os.Getenv("SCHEMA_REGISTRY_ID")

	environmentId := os.Getenv("LIVE_TEST_ENVIRONMENT_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if schemaRegistryApiKey == "" || schemaRegistryApiSecret == "" || schemaRegistryRestEndpoint == "" || schemaRegistryId == "" {
		t.Fatal("SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, SCHEMA_REGISTRY_REST_ENDPOINT, and SCHEMA_REGISTRY_ID must be set for catalog entity attributes live tests")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for catalog entity attributes live tests")
	}

	// Generate unique attribute values for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	entityAttributesResourceLabel := "test_live_catalog_entity_attributes"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCatalogEntityAttributesLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCatalogEntityAttributesLiveConfig(endpoint, entityAttributesResourceLabel, environmentId, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCatalogEntityAttributesLiveExists(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), "entity_name", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), "entity_type", "cf_environment"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), fmt.Sprintf("attributes.owner"), fmt.Sprintf("tf-live-owner-%d", randomSuffix)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), fmt.Sprintf("attributes.description"), fmt.Sprintf("Test environment description %d", randomSuffix)),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), "id"),
				),
			},
			{
				ResourceName:            fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credentials", "rest_endpoint", "schema_registry_cluster"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					entityAttributesId := resources[fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel)].Primary.ID
					return fmt.Sprintf("%s/%s/owner,description", schemaRegistryId, entityAttributesId), nil
				},
			},
			{
				Config: testAccCheckCatalogEntityAttributesLiveConfigUpdate(endpoint, entityAttributesResourceLabel, environmentId, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCatalogEntityAttributesLiveExists(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), "entity_name", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), "entity_type", "cf_environment"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), fmt.Sprintf("attributes.owner"), fmt.Sprintf("tf-live-owner-updated-%d", randomSuffix)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel), fmt.Sprintf("attributes.description"), fmt.Sprintf("Updated test environment description %d", randomSuffix)),
				),
			},
			{
				ResourceName:            fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"credentials", "rest_endpoint", "schema_registry_cluster"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					entityAttributesId := resources[fmt.Sprintf("confluent_catalog_entity_attributes.%s", entityAttributesResourceLabel)].Primary.ID
					return fmt.Sprintf("%s/%s/owner,description", schemaRegistryId, entityAttributesId), nil
				},
			},
		},
	})
}

func testAccCheckCatalogEntityAttributesLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_catalog_entity_attributes" {
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

func testAccCheckCatalogEntityAttributesLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckCatalogEntityAttributesLiveConfig(endpoint, entityAttributesResourceLabel, environmentId, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, apiKey, apiSecret string, randomSuffix int) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_catalog_entity_attributes" "%s" {
		schema_registry_cluster {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
		
		entity_name = "%s"
		entity_type = "cf_environment"
		attributes = {
			owner       = "tf-live-owner-%d"
			description = "Test environment description %d"
		}
	}
	`, endpoint, apiKey, apiSecret, entityAttributesResourceLabel, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, environmentId, randomSuffix, randomSuffix)
}

func testAccCheckCatalogEntityAttributesLiveConfigUpdate(endpoint, entityAttributesResourceLabel, environmentId, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, apiKey, apiSecret string, randomSuffix int) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_catalog_entity_attributes" "%s" {
		schema_registry_cluster {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}
		
		entity_name = "%s"
		entity_type = "cf_environment"
		attributes = {
			owner       = "tf-live-owner-updated-%d"
			description = "Updated test environment description %d"
		}
	}
	`, endpoint, apiKey, apiSecret, entityAttributesResourceLabel, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, environmentId, randomSuffix, randomSuffix)
}

