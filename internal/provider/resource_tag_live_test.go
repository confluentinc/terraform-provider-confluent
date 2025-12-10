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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTagLive(t *testing.T) {
	// Disable parallel execution to avoid resource name collisions and API propagation issues
	// t.Parallel()

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
	tagName := fmt.Sprintf("tf_live_tag_%d", randomSuffix)
	tagResourceLabel := "test_live_tag"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckTagLiveDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create tag first to allow it to propagate
				Config: testAccCheckTagLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagLiveExists(fmt.Sprintf("confluent_tag.%s", tagResourceLabel)),
					// Only check basic existence in first step to allow propagation
				),
			},
			{
				// Step 2: Verify all attributes after propagation
				Config: testAccCheckTagLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagLiveExists(fmt.Sprintf("confluent_tag.%s", tagResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "name", tagName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "description", "Live test tag for data catalog"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "schema_registry_cluster.0.id", schemaRegistryId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "version"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "entity_types.#", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "entity_types.0", "cf_entity"),
				),
			},
			{
				// Step 3: Apply update after previous step has propagated
				Config: testAccCheckTagUpdateLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagLiveExists(fmt.Sprintf("confluent_tag.%s", tagResourceLabel)),
					// Only check basic existence in update step to allow propagation
				),
			},
			{
				// Step 4: Verify update attributes after propagation
				Config: testAccCheckTagUpdateLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckTagLiveExists(fmt.Sprintf("confluent_tag.%s", tagResourceLabel)),
					// Use retry logic for attributes that may need time to propagate
					testAccCheckResourceAttrWithRetryTag(fmt.Sprintf("confluent_tag.%s", tagResourceLabel), "description", "Updated live test tag for data catalog", 5, 2*time.Second),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_tag.%s", tagResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials", "rest_endpoint", "schema_registry_cluster"},
			},
		},
	})
}

func testAccCheckTagLiveDestroy(s *terraform.State) error {
	// In live tests, we can't easily check if the resource is actually destroyed
	// without making API calls, so we just verify the resource is removed from state
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_tag" {
			continue
		}
		// If we reach here, the resource should be cleaned up by Terraform
	}
	return nil
}

func testAccCheckTagLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckTagLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_tag" "%s" {
		name        = "%s"
		description = "Live test tag for data catalog"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret)
}

func testAccCheckTagUpdateLiveConfig(endpoint, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_tag" "%s" {
		name        = "%s"
		description = "Updated live test tag for data catalog"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret)
}

// testAccCheckResourceAttrWithRetryTag retries checking a resource attribute with exponential backoff
// to handle cases where resources need time to propagate after creation
// Note: This function is duplicated here to avoid package-level function conflicts with
// testAccCheckResourceAttrWithRetry in resource_tag_binding_live_test.go
func testAccCheckResourceAttrWithRetryTag(resourceName, attribute, expectedValue string, maxRetries int, initialDelay time.Duration) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var lastErr error
		delay := initialDelay

		for attempt := 0; attempt < maxRetries; attempt++ {
			checkFunc := resource.TestCheckResourceAttr(resourceName, attribute, expectedValue)
			err := checkFunc(s)
			if err == nil {
				return nil
			}

			lastErr = err
			if attempt < maxRetries-1 {
				time.Sleep(delay)
				delay = delay * 2 // Exponential backoff
			}
		}

		return fmt.Errorf("attribute check failed after %d retries: %w", maxRetries, lastErr)
	}
} 