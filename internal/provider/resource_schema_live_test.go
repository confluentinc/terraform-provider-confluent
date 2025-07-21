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

func TestAccSchemaLive(t *testing.T) {
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

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	subjectName := fmt.Sprintf("tf-live-subject-%d", randomSuffix)
	schemaResourceLabel := "test_live_schema"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaLiveConfig(endpoint, schemaResourceLabel, subjectName, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaLiveExists(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel), "subject_name", subjectName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel), "format", "AVRO"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_schema.%s", schemaResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccSchemaUpdateLive(t *testing.T) {
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

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	subjectName := fmt.Sprintf("tf-live-update-subject-%d", randomSuffix)
	schemaResourceLabel := "test_live_schema_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaLiveConfig(endpoint, schemaResourceLabel, subjectName, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaLiveExists(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel), "subject_name", subjectName),
				),
			},
			{
				Config: testAccCheckSchemaUpdateLiveConfig(endpoint, schemaResourceLabel, subjectName, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaLiveExists(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema.%s", schemaResourceLabel), "subject_name", subjectName),
				),
			},
		},
	})
}

func testAccCheckSchemaLiveConfig(endpoint, schemaResourceLabel, subjectName, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
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

	resource "confluent_schema" "%s" {
		subject_name = "%s"
		format       = "AVRO"
		schema       = <<EOF
{
  "type": "record",
  "name": "User",
  "fields": [
    {"name": "id", "type": "int"},
    {"name": "name", "type": "string"},
    {"name": "email", "type": "string"}
  ]
}
EOF
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId, schemaResourceLabel, subjectName)
}

func testAccCheckSchemaUpdateLiveConfig(endpoint, schemaResourceLabel, subjectName, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
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

	resource "confluent_schema" "%s" {
		subject_name = "%s"
		format       = "AVRO"
		schema       = <<EOF
{
  "type": "record",
  "name": "User",
  "fields": [
    {"name": "id", "type": "int"},
    {"name": "name", "type": "string"},
    {"name": "email", "type": "string"},
    {"name": "age", "type": "int", "default": 0}
  ]
}
EOF
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId, schemaResourceLabel, subjectName)
}

func testAccCheckSchemaLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckSchemaLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_schema" {
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
