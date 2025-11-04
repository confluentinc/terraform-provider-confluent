//go:build live_test && (all || schema_registry)

// Copyright 2024 Confluent Inc. All Rights Reserved.
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

func TestAccSchemaRegistryKekLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, SCHEMA_REGISTRY_REST_ENDPOINT, and SCHEMA_REGISTRY_ID must be set for Schema Registry KEK live tests")
	}

	// KEK requires cloud provider KMS credentials
	kmsType := os.Getenv("TEST_KMS_TYPE")       // e.g., "aws-kms", "azure-kms", "gcp-kms"
	kmsKeyId := os.Getenv("TEST_KMS_KEY_ID")     // KMS key ARN or ID
	if kmsType == "" || kmsKeyId == "" {
		t.Skip("TEST_KMS_TYPE and TEST_KMS_KEY_ID environment variables must be set for KEK live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	kekName := fmt.Sprintf("tf-live-kek-%d", randomSuffix)
	kekResourceLabel := "test_live_schema_registry_kek"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryKekLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryKekLiveConfig(endpoint, kekResourceLabel, kekName, kmsType, kmsKeyId, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryKekLiveExists(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "name", kekName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "kms_type", kmsType),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "kms_key_id", kmsKeyId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "shared", "false"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "hard_delete", "false"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "id"),
				),
			},
			{
				Config: testAccCheckSchemaRegistryKekUpdateLiveConfig(endpoint, kekResourceLabel, kekName, kmsType, kmsKeyId, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryKekLiveExists(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "name", kekName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_schema_registry_kek.%s", kekResourceLabel), "doc", "Updated KEK documentation"),
				),
			},
		},
	})
}

func testAccCheckSchemaRegistryKekLiveDestroy(s *terraform.State) error {
	// Note: KEK resources may persist in Schema Registry after terraform destroy if hard_delete is false
	// This is expected behavior for soft delete
	return nil
}

func testAccCheckSchemaRegistryKekLiveConfig(endpoint, resourceLabel, kekName, kmsType, kmsKeyId, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
		schema_registry_id = "%s"
		schema_registry_rest_endpoint = "%s"
		schema_registry_api_key = "%s"
		schema_registry_api_secret = "%s"
	}

	resource "confluent_schema_registry_kek" "%s" {
		name = "%s"
		kms_type = "%s"
		kms_key_id = "%s"
		shared = false
		hard_delete = false
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, resourceLabel, kekName, kmsType, kmsKeyId)
}

func testAccCheckSchemaRegistryKekUpdateLiveConfig(endpoint, resourceLabel, kekName, kmsType, kmsKeyId, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret, schemaRegistryRestEndpoint, schemaRegistryId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
		schema_registry_id = "%s"
		schema_registry_rest_endpoint = "%s"
		schema_registry_api_key = "%s"
		schema_registry_api_secret = "%s"
	}

	resource "confluent_schema_registry_kek" "%s" {
		name = "%s"
		kms_type = "%s"
		kms_key_id = "%s"
		shared = false
		hard_delete = false
		doc = "Updated KEK documentation"
	}
	`, endpoint, apiKey, apiSecret, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, resourceLabel, kekName, kmsType, kmsKeyId)
}

func testAccCheckSchemaRegistryKekLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Schema Registry KEK has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Schema Registry KEK", n)
		}

		return nil
	}
}

