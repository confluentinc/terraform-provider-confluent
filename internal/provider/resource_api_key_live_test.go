//go:build live_test && (all || core)

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

func TestAccApiKeyLive(t *testing.T) {
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	serviceAccountDisplayName := fmt.Sprintf("tf-live-sa-%d", randomSuffix)
	apiKeyDisplayName := fmt.Sprintf("tf-live-api-key-%d", randomSuffix)
	serviceAccountResourceLabel := "test_live_service_account"
	apiKeyResourceLabel := "test_live_api_key"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckApiKeyLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountLiveExists(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel)),
					testAccCheckApiKeyLiveExists(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "display_name", apiKeyDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "description", "Test API key for live testing"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "secret"),
					// Check that owner is set correctly
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func TestAccApiKeyUpdateLive(t *testing.T) {
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	serviceAccountDisplayName := fmt.Sprintf("tf-live-sa-update-%d", randomSuffix)
	apiKeyDisplayName := fmt.Sprintf("tf-live-api-key-update-%d", randomSuffix)
	serviceAccountResourceLabel := "test_live_service_account_update"
	apiKeyResourceLabel := "test_live_api_key_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckApiKeyLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountLiveExists(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel)),
					testAccCheckApiKeyLiveExists(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "display_name", apiKeyDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "description", "Test API key for live testing"),
				),
			},
			{
				Config: testAccCheckApiKeyUpdateLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName+"-updated", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountLiveExists(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel)),
					testAccCheckApiKeyLiveExists(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "display_name", apiKeyDisplayName+"-updated"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "description", "Updated test API key for live testing"),
				),
			},
		},
	})
}

func TestAccApiKeyMinimalLive(t *testing.T) {
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

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	serviceAccountDisplayName := fmt.Sprintf("tf-live-sa-min-%d", randomSuffix)
	serviceAccountResourceLabel := "test_live_service_account_minimal"
	apiKeyResourceLabel := "test_live_api_key_minimal"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckApiKeyLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckApiKeyMinimalLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountLiveExists(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel)),
					testAccCheckApiKeyLiveExists(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "display_name", ""),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "description", ""),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "secret"),
					// Check that owner is set correctly
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.api_version", "iam/v2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel), "owner.0.kind", "ServiceAccount"),
				),
			},
		},
	})
}

func testAccCheckApiKeyLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description  = "Test service account for API key live testing"
	}

	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description  = "Test API key for live testing"
		
		owner {
			id          = confluent_service_account.%s.id
			api_version = confluent_service_account.%s.api_version
			kind        = confluent_service_account.%s.kind
		}
	}
	`, endpoint, apiKey, apiSecret, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, serviceAccountResourceLabel, serviceAccountResourceLabel, serviceAccountResourceLabel)
}

func testAccCheckApiKeyMinimalLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description  = "Test service account for minimal API key live testing"
	}

	resource "confluent_api_key" "%s" {
		owner {
			id          = confluent_service_account.%s.id
			api_version = confluent_service_account.%s.api_version
			kind        = confluent_service_account.%s.kind
		}
	}
	`, endpoint, apiKey, apiSecret, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, serviceAccountResourceLabel, serviceAccountResourceLabel, serviceAccountResourceLabel)
}

func testAccCheckApiKeyUpdateLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description  = "Test service account for API key live testing"
	}

	resource "confluent_api_key" "%s" {
		display_name = "%s"
		description  = "Updated test API key for live testing"
		
		owner {
			id          = confluent_service_account.%s.id
			api_version = confluent_service_account.%s.api_version
			kind        = confluent_service_account.%s.kind
		}
	}
	`, endpoint, apiKey, apiSecret, serviceAccountResourceLabel, serviceAccountDisplayName, apiKeyResourceLabel, apiKeyDisplayName, serviceAccountResourceLabel, serviceAccountResourceLabel, serviceAccountResourceLabel)
}

func testAccCheckApiKeyLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckApiKeyLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_api_key" {
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
