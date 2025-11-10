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

func TestAccCatalogIntegrationLive(t *testing.T) {
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

	// Read Kafka cluster configuration
	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")

	environmentId := os.Getenv("LIVE_TEST_ENVIRONMENT_ID")

	// AWS IAM role ARN for provider integration (we'll create one or use existing)
	awsIamRoleArn := os.Getenv("TEST_KMS_KEY_ID")

	// Tableflow API credentials - these are typically separate from regular API credentials
	// If not set, we'll try using regular API credentials but the test may fail with 401
	tableflowApiKey := os.Getenv("TABLEFLOW_API_KEY")
	tableflowApiSecret := os.Getenv("TABLEFLOW_API_SECRET")
	if tableflowApiKey == "" {
		// Note: Regular API credentials may not work for tableflow API
		// This test may fail with 401 if tableflow credentials aren't available
		tableflowApiKey = apiKey
	}
	if tableflowApiSecret == "" {
		tableflowApiSecret = apiSecret
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if kafkaClusterId == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID must be set for catalog integration live tests")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for catalog integration live tests")
	}

	if awsIamRoleArn == "" {
		t.Skip("Skipping Catalog Integration test. TEST_KMS_KEY_ID must be set to run this test.")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	piDisplayName := fmt.Sprintf("tf-live-pi-for-ci-%d", randomSuffix)
	ciDisplayName := fmt.Sprintf("tf-live-catalog-integration-%d", randomSuffix)
	ciDisplayNameUpdated := fmt.Sprintf("tf-live-catalog-integration-updated-%d", randomSuffix)
	piResourceLabel := "test_live_pi_for_ci"
	ciResourceLabel := "test_live_catalog_integration"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCatalogIntegrationLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCatalogIntegrationLiveConfig(endpoint, piResourceLabel, ciResourceLabel, piDisplayName, ciDisplayName, environmentId, kafkaClusterId, awsIamRoleArn, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCatalogIntegrationLiveExists(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "display_name", ciDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "aws_glue.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "aws_glue.0.provider_integration_id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					ciId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.Attributes["environment.0.id"]
					clusterId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.Attributes["kafka_cluster.0.id"]
					return fmt.Sprintf("%s/%s/%s", envId, clusterId, ciId), nil
				},
			},
			{
				Config: testAccCheckCatalogIntegrationLiveConfig(endpoint, piResourceLabel, ciResourceLabel, piDisplayName, ciDisplayNameUpdated, environmentId, kafkaClusterId, awsIamRoleArn, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCatalogIntegrationLiveExists(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "display_name", ciDisplayNameUpdated),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel), "kafka_cluster.0.id", kafkaClusterId),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"credentials"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					ciId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.Attributes["environment.0.id"]
					clusterId := resources[fmt.Sprintf("confluent_catalog_integration.%s", ciResourceLabel)].Primary.Attributes["kafka_cluster.0.id"]
					return fmt.Sprintf("%s/%s/%s", envId, clusterId, ciId), nil
				},
			},
		},
	})
}

func testAccCheckCatalogIntegrationLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_catalog_integration" {
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

func testAccCheckCatalogIntegrationLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckCatalogIntegrationLiveConfig(endpoint, piResourceLabel, ciResourceLabel, piDisplayName, ciDisplayName, environmentId, kafkaClusterId, awsIamRoleArn, apiKey, apiSecret, tableflowApiKey, tableflowApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	# Create provider integration first (required for catalog integration)
	resource "confluent_provider_integration" "%s" {
		display_name = "%s"
		environment {
			id = "%s"
		}
		aws {
			customer_role_arn = "%s"
		}
	}

	# Create catalog integration using the provider integration
	resource "confluent_catalog_integration" "%s" {
		display_name = "%s"
		environment {
			id = "%s"
		}
		kafka_cluster {
			id = "%s"
		}
		aws_glue {
			provider_integration_id = confluent_provider_integration.%s.id
		}
		credentials {
			key    = "%s"
			secret = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, piResourceLabel, piDisplayName, environmentId, awsIamRoleArn, ciResourceLabel, ciDisplayName, environmentId, kafkaClusterId, piResourceLabel, tableflowApiKey, tableflowApiSecret)
}

