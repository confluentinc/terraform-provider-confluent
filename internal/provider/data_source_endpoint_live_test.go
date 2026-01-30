//go:build live_test && (all || networking)

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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccEndpointDataSourceLive(t *testing.T) {
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

	// Skip test if required environment variables are not set
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Get test environment ID from environment variable or use default
	testEnvironmentId := os.Getenv("CONFLUENT_TEST_ENVIRONMENT_ID")
	if testEnvironmentId == "" {
		testEnvironmentId = "env-stgcg7md23" // Default test environment
	}

	endpointDataSourceLabel := "test_live_endpoint"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEndpointDataSourceLiveConfig(endpoint, endpointDataSourceLabel, apiKey, apiSecret, testEnvironmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointDataSourceLiveExists(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel)),
					// Verify endpoints list exists and has at least one endpoint
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.#"),
					// Verify first endpoint has required computed fields
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.api_version"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.kind"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.service"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.cloud"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.region"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.endpoint_type"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.connection_type"),
					// Verify environment block is populated
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.environment.0.id", testEnvironmentId),
				),
			},
		},
	})
}

func TestAccEndpointDataSourceLiveWithFilters(t *testing.T) {
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

	// Skip test if required environment variables are not set
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Get test environment ID from environment variable or use default
	testEnvironmentId := os.Getenv("CONFLUENT_TEST_ENVIRONMENT_ID")
	if testEnvironmentId == "" {
		testEnvironmentId = "env-stgcg7md23" // Default test environment
	}

	endpointDataSourceLabel := "test_live_endpoint_filtered"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEndpointDataSourceLiveConfigWithFilters(endpoint, endpointDataSourceLabel, apiKey, apiSecret, testEnvironmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointDataSourceLiveExists(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel)),
					// Verify endpoints list exists
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.#"),
					// Verify filters are applied - check that returned endpoints match filter criteria
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.service", "KAFKA"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.is_private", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.environment.0.id", testEnvironmentId),
				),
			},
		},
	})
}

func TestAccEndpointDataSourceLiveWithResourceFilter(t *testing.T) {
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

	// Skip test if required environment variables are not set
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Get test environment ID and resource ID from environment variables or use defaults
	testEnvironmentId := os.Getenv("CONFLUENT_TEST_ENVIRONMENT_ID")
	if testEnvironmentId == "" {
		testEnvironmentId = "env-stgcg7md23" // Default test environment
	}
	testResourceId := os.Getenv("CONFLUENT_TEST_KAFKA_CLUSTER_ID")
	if testResourceId == "" {
		// If no cluster ID is provided, skip this test
		t.Skip("CONFLUENT_TEST_KAFKA_CLUSTER_ID must be set to test resource filter")
	}

	endpointDataSourceLabel := "test_live_endpoint_with_resource"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEndpointDataSourceLiveConfigWithResource(endpoint, endpointDataSourceLabel, apiKey, apiSecret, testEnvironmentId, testResourceId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointDataSourceLiveExists(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel)),
					// Verify endpoints list exists
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.#"),
					// Verify resource filter is applied - check that returned endpoints have the correct resource
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.service", "KAFKA"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.resource.#", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_endpoint.%s", endpointDataSourceLabel), "endpoints.0.resource.0.id", testResourceId),
				),
			},
		},
	})
}

func testAccCheckEndpointDataSourceLiveConfig(endpoint, endpointDataSourceLabel, apiKey, apiSecret, environmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint          = "%s"
		cloud_api_key     = "%s"
		cloud_api_secret  = "%s"
	}

	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "KAFKA"
		}
	}
	`, endpoint, apiKey, apiSecret, endpointDataSourceLabel, environmentId)
}

func testAccCheckEndpointDataSourceLiveConfigWithFilters(endpoint, endpointDataSourceLabel, apiKey, apiSecret, environmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint          = "%s"
		cloud_api_key     = "%s"
		cloud_api_secret  = "%s"
	}

	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "KAFKA"
			cloud = "AWS"
			region = "us-west-2"
			is_private = true
		}
	}
	`, endpoint, apiKey, apiSecret, endpointDataSourceLabel, environmentId)
}

func testAccCheckEndpointDataSourceLiveConfigWithResource(endpoint, endpointDataSourceLabel, apiKey, apiSecret, environmentId, resourceId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint          = "%s"
		cloud_api_key     = "%s"
		cloud_api_secret  = "%s"
	}

	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "KAFKA"
			resource = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, endpointDataSourceLabel, environmentId, resourceId)
}

func testAccCheckEndpointDataSourceLiveExists(resourceName string) resource.TestCheckFunc {
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
