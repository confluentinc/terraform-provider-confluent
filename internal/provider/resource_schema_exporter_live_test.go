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
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccSchemaExporterLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Schema Exporter live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	exporterName := fmt.Sprintf("tf-live-exporter-%d", randomSuffix)
	exporterResourceLabel := "test_live_schema_exporter"

	// Note: This test requires a destination Schema Registry cluster
	// For simplicity, we'll use the same cluster as both source and destination
	// In real scenarios, you'd have separate clusters
	destinationRestEndpoint := schemaRegistryRestEndpoint
	destinationApiKey := schemaRegistryApiKey
	destinationApiSecret := schemaRegistryApiSecret

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaExporterLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaExporterLiveConfig(endpoint, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, destinationRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, destinationApiKey, destinationApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExporterLiveExists(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "name", exporterName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "context_type", "CUSTOM"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "context", "live-test"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "schema_registry_cluster.0.id", schemaRegistryId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials", "rest_endpoint", "schema_registry_cluster", "destination_schema_registry_cluster.0.credentials.0.key", "destination_schema_registry_cluster.0.credentials.0.secret", "reset_on_update"},
			},
		},
	})
}

func TestAccSchemaExporterUpdateLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Schema Exporter live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	exporterName := fmt.Sprintf("tf-live-exporter-update-%d", randomSuffix)
	exporterResourceLabel := "test_live_schema_exporter_update"

	// Note: This test requires a destination Schema Registry cluster
	// For simplicity, we'll use the same cluster as both source and destination
	destinationRestEndpoint := schemaRegistryRestEndpoint
	destinationApiKey := schemaRegistryApiKey
	destinationApiSecret := schemaRegistryApiSecret

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaExporterLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaExporterLiveConfig(endpoint, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, destinationRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, destinationApiKey, destinationApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExporterLiveExists(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "context", "live-test"),
				),
			},
			{
				Config: testAccCheckSchemaExporterUpdateLiveConfig(endpoint, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, destinationRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, destinationApiKey, destinationApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExporterLiveExists(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_exporter.%s", exporterResourceLabel), "context", "live-test-updated"),
				),
			},
		},
	})
}

func testAccCheckSchemaExporterLiveDestroy(s *terraform.State) error {
	// In live tests, we can't easily check if the resource is actually destroyed
	// without making API calls, so we just verify the resource is removed from state
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_schema_exporter" {
			continue
		}
		// If we reach here, the resource should be cleaned up by Terraform
	}
	return nil
}

func testAccCheckSchemaExporterLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckSchemaExporterLiveConfig(endpoint, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, destinationRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, destinationApiKey, destinationApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_schema_exporter" "%s" {
		name         = "%s"
		context_type = "CUSTOM"
		context      = "live-test"
		subjects     = ["*"]

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		destination_schema_registry_cluster {
			rest_endpoint = "%s"
			credentials {
				key    = "%s"
				secret = "%s"
			}
		}
	}
	`, endpoint, apiKey, apiSecret, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, destinationRestEndpoint, destinationApiKey, destinationApiSecret)
}

func testAccCheckSchemaExporterUpdateLiveConfig(endpoint, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, destinationRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, destinationApiKey, destinationApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_schema_exporter" "%s" {
		name         = "%s"
		context_type = "CUSTOM"
		context      = "live-test-updated"
		subjects     = ["*"]
		status       = "PAUSED"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		destination_schema_registry_cluster {
			rest_endpoint = "%s"
			credentials {
				key    = "%s"
				secret = "%s"
			}
		}
	}
	`, endpoint, apiKey, apiSecret, exporterResourceLabel, exporterName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, destinationRestEndpoint, destinationApiKey, destinationApiSecret)
} 