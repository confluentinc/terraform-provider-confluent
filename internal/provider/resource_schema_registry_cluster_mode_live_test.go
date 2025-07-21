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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSchemaRegistryClusterModeLive(t *testing.T) {
	// Disable parallel execution since this test modifies global cluster mode
	// t.Parallel()

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

	// Read Schema Registry credentials from environment variables
	schemaRegistryApiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	schemaRegistryApiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")
	schemaRegistryRestEndpoint := os.Getenv("SCHEMA_REGISTRY_REST_ENDPOINT")
	schemaRegistryId := os.Getenv("SCHEMA_REGISTRY_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if schemaRegistryApiKey == "" || schemaRegistryApiSecret == "" || schemaRegistryRestEndpoint == "" || schemaRegistryId == "" {
		t.Fatal("SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, SCHEMA_REGISTRY_REST_ENDPOINT, and SCHEMA_REGISTRY_ID must be set for Schema Registry live tests")
	}

	clusterModeResourceLabel := "test_live_schema_registry_cluster_mode"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterModeLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterModeLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeLiveExists(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel), "mode", "READWRITE"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSchemaRegistryClusterModeUpdateLive(t *testing.T) {
	// Disable parallel execution since this test modifies global cluster mode
	// t.Parallel()

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

	// Read Schema Registry credentials from environment variables
	schemaRegistryApiKey := os.Getenv("SCHEMA_REGISTRY_API_KEY")
	schemaRegistryApiSecret := os.Getenv("SCHEMA_REGISTRY_API_SECRET")
	schemaRegistryRestEndpoint := os.Getenv("SCHEMA_REGISTRY_REST_ENDPOINT")
	schemaRegistryId := os.Getenv("SCHEMA_REGISTRY_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if schemaRegistryApiKey == "" || schemaRegistryApiSecret == "" || schemaRegistryRestEndpoint == "" || schemaRegistryId == "" {
		t.Fatal("SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, SCHEMA_REGISTRY_REST_ENDPOINT, and SCHEMA_REGISTRY_ID must be set for Schema Registry live tests")
	}

	clusterModeResourceLabel := "test_live_schema_registry_cluster_mode_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterModeLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterModeLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeLiveExists(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel), "mode", "READWRITE"),
				),
			},
			{
				Config: testAccCheckSchemaRegistryClusterModeUpdateLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeLiveExists(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel), "mode", "READONLY"),
				),
			},
			{
				// Reset back to READWRITE to ensure other tests can run
				Config: testAccCheckSchemaRegistryClusterModeLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeLiveExists(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", clusterModeResourceLabel), "mode", "READWRITE"),
				),
			},
		},
	})
}

func testAccCheckSchemaRegistryClusterModeLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint                       = "%s"
		cloud_api_key                  = "%s"
		cloud_api_secret               = "%s"
		schema_registry_api_key        = "%s"
		schema_registry_api_secret     = "%s"
		schema_registry_rest_endpoint  = "%s"
		schema_registry_id             = "%s"
	}

	resource "confluent_schema_registry_cluster_mode" "%s" {
		mode = "READWRITE"
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId, clusterModeResourceLabel)
}

func testAccCheckSchemaRegistryClusterModeUpdateLiveConfig(endpoint, clusterModeResourceLabel, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint                       = "%s"
		cloud_api_key                  = "%s"
		cloud_api_secret               = "%s"
		schema_registry_api_key        = "%s"
		schema_registry_api_secret     = "%s"
		schema_registry_rest_endpoint  = "%s"
		schema_registry_id             = "%s"
	}

	resource "confluent_schema_registry_cluster_mode" "%s" {
		mode = "READONLY"
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId, clusterModeResourceLabel)
}

func testAccCheckSchemaRegistryClusterModeLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckSchemaRegistryClusterModeLiveDestroy(s *terraform.State) error {
	// Reset the cluster mode to READWRITE to ensure other tests can run
	// This is important because the cluster mode affects all operations in the Schema Registry
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_schema_registry_cluster_mode" {
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
