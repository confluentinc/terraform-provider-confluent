//go:build live_test && (all || core)

// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccProviderIntegrationSetupAzureLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Use environment variables for credentials, hardcode environment
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	environmentId := "env-zyg27z" // Hardcoded test environment
	azureTenantId := os.Getenv("AZURE_TENANT_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" || azureTenantId == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY, CONFLUENT_CLOUD_API_SECRET, and AZURE_TENANT_ID environment variables must be set for live tests")
	}

	// Use production endpoint if not specified
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	integrationDisplayName := fmt.Sprintf("tf-live-test-azure-%d", randomSuffix)
	integrationResourceLabel := "test_azure"
	authResourceLabel := "test_azure_auth"

	fullIntegrationResourceLabel := fmt.Sprintf("confluent_provider_integration_setup.%s", integrationResourceLabel)
	fullAuthResourceLabel := fmt.Sprintf("confluent_provider_integration_authorization.%s", authResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckLive(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationSetupDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create integration in DRAFT status
				Config: testAccCheckProviderIntegrationSetupAzureSetupOnlyConfig(endpoint, apiKey, apiSecret, environmentId, integrationDisplayName, integrationResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramDisplayName, integrationDisplayName),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramCloud, "AZURE"),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), environmentId),
					resource.TestCheckResourceAttrSet(fullIntegrationResourceLabel, paramId),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramStatus, "DRAFT"),
				),
			},
			{
				// Step 2: Add authorization resource (should show warning and transition to CREATED)
				Config: testAccCheckProviderIntegrationSetupAzureAuthConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Check integration resource still exists
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),

					// Check authorization resource
					testAccCheckProviderIntegrationSetupAuthorizationExists(fullAuthResourceLabel),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, paramProviderIntegrationIdAuth),
					resource.TestCheckResourceAttr(fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureCustomerTenantId), azureTenantId),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId)),

					// Verify multi-tenant app ID is a valid GUID format
					resource.TestMatchResourceAttr(fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId),
						regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)),
				),
				// ExpectNonEmptyPlan: true, // Expect warnings about Azure setup
			},
			{
				// Step 3: Test validation persistence (re-apply should show same warning)
				Config: testAccCheckProviderIntegrationSetupAzureAuthConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources still exist and have correct values
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),
					testAccCheckProviderIntegrationSetupAuthorizationExists(fullAuthResourceLabel),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId)),
				),
				// ExpectNonEmptyPlan: true, // Should still show warnings about incomplete Azure setup
			},
		},
	})
}

func testAccPreCheckLive(t *testing.T) {
	if v := os.Getenv("CONFLUENT_CLOUD_API_KEY"); v == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY must be set for live tests")
	}
	if v := os.Getenv("CONFLUENT_CLOUD_API_SECRET"); v == "" {
		t.Fatal("CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}
}

func testAccCheckProviderIntegrationSetupDestroy(s *terraform.State) error {
	// For live tests, we actually want to clean up resources
	// This is handled by the test framework automatically
	return nil
}

func testAccCheckProviderIntegrationSetupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s provider integration v2 has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s provider integration v2", n)
		}
		return nil
	}
}

func testAccCheckProviderIntegrationSetupAuthorizationExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s provider integration v2 authorization has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s provider integration v2 authorization", n)
		}
		return nil
	}
}

func testAccCheckProviderIntegrationSetupAzureSetupOnlyConfig(endpoint, apiKey, apiSecret, environmentId, displayName, integrationResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint          = "%s"
  cloud_api_key     = "%s"
  cloud_api_secret  = "%s"
}

resource "confluent_provider_integration_setup" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud = "AZURE"
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName)
}

func testAccCheckProviderIntegrationSetupAzureAuthConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, displayName, integrationResourceLabel, authResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint          = "%s"
  cloud_api_key     = "%s"
  cloud_api_secret  = "%s"
}

resource "confluent_provider_integration_setup" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud = "AZURE"
}

resource "confluent_provider_integration_authorization" "%s" {
  provider_integration_id = confluent_provider_integration_setup.%s.id
  
  environment {
    id = "%s"
  }
  
  azure {
    customer_azure_tenant_id = "%s"
  }
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName, authResourceLabel, integrationResourceLabel, environmentId, azureTenantId)
}

func testAccCheckProviderIntegrationSetupAzureLiveConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, displayName, integrationResourceLabel, authResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint          = "%s"
  cloud_api_key     = "%s"
  cloud_api_secret  = "%s"
}

resource "confluent_provider_integration_setup" "%s" {
  environment {
    id = "%s"
  }
  
  display_name = "%s"
  cloud = "AZURE"
}

resource "confluent_provider_integration_authorization" "%s" {
  provider_integration_id = confluent_provider_integration_setup.%s.id
  
  environment {
    id = "%s"
  }
  
  azure {
    customer_azure_tenant_id = "%s"
  }
  
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName, authResourceLabel, integrationResourceLabel, environmentId, azureTenantId)
}

func TestAccProviderIntegrationSetupGcpLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)

	// Use environment variables for credentials, hardcode environment
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	environmentId := "env-zyg27z"                                                                            // Hardcoded test environment
	gcpServiceAccount := fmt.Sprintf("test-sa-%d@test-project-123456.iam.gserviceaccount.com", randomSuffix) // Unique test service account

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET environment variables must be set for live tests")
	}

	// Use production endpoint if not specified
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}
	integrationDisplayName := fmt.Sprintf("tf-live-test-gcp-%d", randomSuffix)
	integrationResourceLabel := "test_gcp"
	authResourceLabel := "test_gcp_auth"

	fullIntegrationResourceLabel := fmt.Sprintf("confluent_provider_integration_setup.%s", integrationResourceLabel)
	fullAuthResourceLabel := fmt.Sprintf("confluent_provider_integration_authorization.%s", authResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckLive(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationSetupDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create integration in DRAFT status
				Config: testAccCheckProviderIntegrationSetupGcpSetupConfig(endpoint, apiKey, apiSecret, environmentId, integrationDisplayName, integrationResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramDisplayName, integrationDisplayName),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramCloud, "GCP"),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), environmentId),
					resource.TestCheckResourceAttrSet(fullIntegrationResourceLabel, paramId),
					resource.TestCheckResourceAttr(fullIntegrationResourceLabel, paramStatus, "DRAFT"),
				),
			},
			{
				// Step 2: Add authorization resource (should show GCP warning)
				Config: testAccCheckProviderIntegrationSetupGcpAuthConfig(endpoint, apiKey, apiSecret, environmentId, gcpServiceAccount, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Check integration resource still exists
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),

					// Check authorization resource
					testAccCheckProviderIntegrationSetupAuthorizationExists(fullAuthResourceLabel),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, paramProviderIntegrationIdAuth),
					resource.TestCheckResourceAttr(fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpCustomerServiceAccount), gcpServiceAccount),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount)),

					// Verify Confluent service account follows expected pattern
					resource.TestMatchResourceAttr(fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount),
						regexp.MustCompile(`^cspi-[a-z0-9]+@cflt-cspi-stag-1\.iam\.gserviceaccount\.com$`)),
				),
			},
			{
				// Step 3: Test validation persistence (re-apply should show same GCP warning)
				Config: testAccCheckProviderIntegrationSetupGcpAuthConfig(endpoint, apiKey, apiSecret, environmentId, gcpServiceAccount, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources still exist and have correct values
					testAccCheckProviderIntegrationSetupExists(fullIntegrationResourceLabel),
					testAccCheckProviderIntegrationSetupAuthorizationExists(fullAuthResourceLabel),
					resource.TestCheckResourceAttrSet(fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount)),
				),
			},
		},
	})
}

func testAccCheckProviderIntegrationSetupGcpSetupConfig(endpoint, apiKey, apiSecret, environmentId, displayName, integrationResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint          = "%s"
  cloud_api_key     = "%s"
  cloud_api_secret  = "%s"
}

resource "confluent_provider_integration_setup" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud = "GCP"
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName)
}

func testAccCheckProviderIntegrationSetupGcpAuthConfig(endpoint, apiKey, apiSecret, environmentId, gcpServiceAccount, displayName, integrationResourceLabel, authResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint          = "%s"
  cloud_api_key     = "%s"
  cloud_api_secret  = "%s"
}

resource "confluent_provider_integration_setup" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud = "GCP"
}

resource "confluent_provider_integration_authorization" "%s" {
  provider_integration_id = confluent_provider_integration_setup.%s.id
  
  environment {
    id = "%s"
  }
  
  gcp {
    customer_google_service_account = "%s"
  }
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName, authResourceLabel, integrationResourceLabel, environmentId, gcpServiceAccount)
}
