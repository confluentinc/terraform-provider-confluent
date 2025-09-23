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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccProviderIntegrationV2DataSourceLive(t *testing.T) {
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
	integrationDisplayName := fmt.Sprintf("tf-live-test-ds-azure-%d", randomSuffix)
	integrationResourceLabel := "test_azure"
	authResourceLabel := "test_azure_auth"

	fullIntegrationResourceLabel := fmt.Sprintf("confluent_provider_integration_v2.%s", integrationResourceLabel)
	fullAuthResourceLabel := fmt.Sprintf("confluent_provider_integration_v2_authorization.%s", authResourceLabel)
	fullIntegrationDataSourceLabel := fmt.Sprintf("data.confluent_provider_integration_v2.%s", integrationResourceLabel)
	fullAuthDataSourceLabel := fmt.Sprintf("data.confluent_provider_integration_v2_authorization.%s", authResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckLive(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationV2Destroy,
		Steps: []resource.TestStep{
			{
				// Create integration and authorization resources, then test data sources
				Config: testAccCheckProviderIntegrationV2DataSourceLiveConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources exist
					testAccCheckProviderIntegrationV2Exists(fullIntegrationResourceLabel),
					testAccCheckProviderIntegrationV2AuthorizationExists(fullAuthResourceLabel),
					
					// Test integration data source
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramId, fullIntegrationResourceLabel, paramId),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramDisplayName, fullIntegrationResourceLabel, paramDisplayName),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramCloudProvider, fullIntegrationResourceLabel, paramCloudProvider),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramStatus, fullIntegrationResourceLabel, paramStatus),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), fullIntegrationResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId)),
					
					// Test authorization data source
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, paramId, fullAuthResourceLabel, paramId),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, paramProviderIntegrationIdAuth, fullAuthResourceLabel, paramProviderIntegrationIdAuth),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, fmt.Sprintf("azure.0.%s", paramAzureCustomerTenantId), fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureCustomerTenantId)),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId), fullAuthResourceLabel, fmt.Sprintf("azure.0.%s", paramAzureConfluentMultiTenantAppId)),
				),
			},
		},
	})
}

func TestAccProviderIntegrationV2GcpDataSourceLive(t *testing.T) {
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
	environmentId := "env-zyg27z" // Hardcoded test environment
	gcpServiceAccount := fmt.Sprintf("test-sa-%d@test-project-123456.iam.gserviceaccount.com", randomSuffix) // Unique test service account

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET environment variables must be set for live tests")
	}
	
	// Use production endpoint if not specified
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	integrationDisplayName := fmt.Sprintf("tf-live-test-ds-gcp-%d", randomSuffix)
	integrationResourceLabel := "test_gcp"
	authResourceLabel := "test_gcp_auth"

	fullIntegrationResourceLabel := fmt.Sprintf("confluent_provider_integration_v2.%s", integrationResourceLabel)
	fullAuthResourceLabel := fmt.Sprintf("confluent_provider_integration_v2_authorization.%s", authResourceLabel)
	fullIntegrationDataSourceLabel := fmt.Sprintf("data.confluent_provider_integration_v2.%s", integrationResourceLabel)
	fullAuthDataSourceLabel := fmt.Sprintf("data.confluent_provider_integration_v2_authorization.%s", authResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckLive(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationV2Destroy,
		Steps: []resource.TestStep{
			{
				// Create integration and authorization resources, then test data sources
				Config: testAccCheckProviderIntegrationV2GcpDataSourceLiveConfig(endpoint, apiKey, apiSecret, environmentId, gcpServiceAccount, integrationDisplayName, integrationResourceLabel, authResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					// Verify resources exist
					testAccCheckProviderIntegrationV2Exists(fullIntegrationResourceLabel),
					testAccCheckProviderIntegrationV2AuthorizationExists(fullAuthResourceLabel),
					
					// Test integration data source
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramId, fullIntegrationResourceLabel, paramId),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramDisplayName, fullIntegrationResourceLabel, paramDisplayName),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramCloudProvider, fullIntegrationResourceLabel, paramCloudProvider),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, paramStatus, fullIntegrationResourceLabel, paramStatus),
					resource.TestCheckResourceAttrPair(fullIntegrationDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), fullIntegrationResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId)),
					
					// Test authorization data source
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, paramId, fullAuthResourceLabel, paramId),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, paramProviderIntegrationIdAuth, fullAuthResourceLabel, paramProviderIntegrationIdAuth),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpCustomerServiceAccount), fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpCustomerServiceAccount)),
					resource.TestCheckResourceAttrPair(fullAuthDataSourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount), fullAuthResourceLabel, fmt.Sprintf("gcp.0.%s", paramGcpGoogleServiceAccount)),
				),
			},
		},
	})
}

func testAccCheckProviderIntegrationV2DataSourceLiveConfig(endpoint, apiKey, apiSecret, environmentId, azureTenantId, displayName, integrationResourceLabel, authResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint         = "%s"
  cloud_api_key    = "%s"
  cloud_api_secret = "%s"
}

resource "confluent_provider_integration_v2" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud_provider = "azure"
}

resource "confluent_provider_integration_v2_authorization" "%s" {
  provider_integration_id = confluent_provider_integration_v2.%s.id
  
  environment {
    id = "%s"
  }
  
  azure {
    customer_azure_tenant_id = "%s"
  }
}

data "confluent_provider_integration_v2" "%s" {
  id = confluent_provider_integration_v2.%s.id
  environment {
    id = "%s"
  }
}

data "confluent_provider_integration_v2_authorization" "%s" {
  id = confluent_provider_integration_v2_authorization.%s.id
  environment {
    id = "%s"
  }
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName, authResourceLabel, integrationResourceLabel, environmentId, azureTenantId, integrationResourceLabel, integrationResourceLabel, environmentId, authResourceLabel, authResourceLabel, environmentId)
}

func testAccCheckProviderIntegrationV2GcpDataSourceLiveConfig(endpoint, apiKey, apiSecret, environmentId, gcpServiceAccount, displayName, integrationResourceLabel, authResourceLabel string) string {
	return fmt.Sprintf(`
provider "confluent" {
  endpoint         = "%s"
  cloud_api_key    = "%s"
  cloud_api_secret = "%s"
}

resource "confluent_provider_integration_v2" "%s" {
  environment {
    id = "%s"
  }
  
  display_name   = "%s"
  cloud_provider = "gcp"
}

resource "confluent_provider_integration_v2_authorization" "%s" {
  provider_integration_id = confluent_provider_integration_v2.%s.id
  
  environment {
    id = "%s"
  }
  
  gcp {
    customer_google_service_account = "%s"
  }
}

data "confluent_provider_integration_v2" "%s" {
  id = confluent_provider_integration_v2.%s.id
  environment {
    id = "%s"
  }
}

data "confluent_provider_integration_v2_authorization" "%s" {
  id = confluent_provider_integration_v2_authorization.%s.id
  environment {
    id = "%s"
  }
}
`, endpoint, apiKey, apiSecret, integrationResourceLabel, environmentId, displayName, authResourceLabel, integrationResourceLabel, environmentId, gcpServiceAccount, integrationResourceLabel, integrationResourceLabel, environmentId, authResourceLabel, authResourceLabel, environmentId)
}