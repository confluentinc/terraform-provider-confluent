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

func TestAccTagBindingDataSourceLive(t *testing.T) {
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
		t.Fatal("SCHEMA_REGISTRY_ID, SCHEMA_REGISTRY_API_KEY, SCHEMA_REGISTRY_API_SECRET, and SCHEMA_REGISTRY_REST_ENDPOINT must be set for Tag Binding data source live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	tagName := fmt.Sprintf("tf_live_tag_bind_ds_%d", randomSuffix)
	subjectName := fmt.Sprintf("tf-live-subject-bind-ds-%d", randomSuffix)
	tagResourceLabel := "test_live_tag"
	schemaResourceLabel := "test_live_schema"
	tagBindingResourceLabel := "test_live_tag_binding"
	tagBindingDataSourceLabel := "test_live_tag_binding_data"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckTagBindingDataSourceLiveConfig(endpoint, tagResourceLabel, schemaResourceLabel, tagBindingResourceLabel, tagBindingDataSourceLabel, tagName, subjectName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_tag_binding.%s", tagBindingDataSourceLabel), "tag_name", tagName),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_tag_binding.%s", tagBindingDataSourceLabel), "entity_type", "sr_schema"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_tag_binding.%s", tagBindingDataSourceLabel), "schema_registry_cluster.0.id", schemaRegistryId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_tag_binding.%s", tagBindingDataSourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_tag_binding.%s", tagBindingDataSourceLabel), "entity_name"),
				),
			},
		},
	})
}

func testAccCheckTagBindingDataSourceLiveConfig(endpoint, tagResourceLabel, schemaResourceLabel, tagBindingResourceLabel, tagBindingDataSourceLabel, tagName, subjectName, schemaRegistryId, schemaRegistryRestEndpoint, apiKey, apiSecret, schemaRegistryApiKey, schemaRegistryApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Create a tag to bind to the schema
	resource "confluent_tag" "%s" {
		name        = "%s"
		description = "Live test tag for binding data source"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	# Create a schema to bind the tag to
	resource "confluent_schema" "%s" {
		subject_name = "%s"
		format       = "AVRO"
		schema       = jsonencode({
			type = "record"
			name = "User"
			fields = [
				{
					name = "id"
					type = "int"
				},
				{
					name = "name"
					type = "string"
				}
			]
		})

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}
	}

	# Bind the tag to the schema
	resource "confluent_tag_binding" "%s" {
		tag_name    = confluent_tag.%s.name
		entity_name = "${confluent_schema.%s.schema_registry_cluster.0.id}:.:${confluent_schema.%s.schema_identifier}"
		entity_type = "sr_schema"

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		depends_on = [confluent_tag.%s, confluent_schema.%s]
	}

	# Read the tag binding using data source
	data "confluent_tag_binding" "%s" {
		tag_name    = confluent_tag_binding.%s.tag_name
		entity_name = confluent_tag_binding.%s.entity_name
		entity_type = confluent_tag_binding.%s.entity_type

		schema_registry_cluster {
			id = "%s"
		}

		rest_endpoint = "%s"

		credentials {
			key    = "%s"
			secret = "%s"
		}

		depends_on = [confluent_tag_binding.%s]
	}
	`, endpoint, apiKey, apiSecret, tagResourceLabel, tagName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, schemaResourceLabel, subjectName, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, tagBindingResourceLabel, tagResourceLabel, schemaResourceLabel, schemaResourceLabel, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, tagResourceLabel, schemaResourceLabel, tagBindingDataSourceLabel, tagBindingResourceLabel, tagBindingResourceLabel, tagBindingResourceLabel, schemaRegistryId, schemaRegistryRestEndpoint, schemaRegistryApiKey, schemaRegistryApiSecret, tagBindingResourceLabel)
} 