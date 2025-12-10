//go:build live_test && (all || rbac || drift)

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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccIdentityProviderDriftDetection is a drift detection test that validates the provider's
// ability to accurately read and represent identity provider resources from Confluent Cloud.
//
// Unlike standard live tests that create and destroy resources, this test:
// - Uses a persistent, manually created identity provider resource
// - Reads the resource via data source (ensuring no resource destruction)
// - Validates that all attributes match expected values and detects drift
//
// Test Behavior:
// - PASS: Provider correctly reads all attributes; no drift detected
// - FAIL: Attributes don't match (drift detected)
func TestAccIdentityProviderDriftDetection(t *testing.T) {
	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping drift detection test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for drift detection tests")
	}

	// ========================================
	// CONFIGURATION: Update these values to match your manually created resources
	// ========================================
	// Resource ID for the pre-existing identity provider
	identityProviderId := "op-9N9A"

	// Expected attribute values - must match the actual resource in Confluent Cloud
	// If these don't match, the test will fail, indicating drift
	expectedDisplayName := "Drift Detection Test - Identity Provider"
	expectedDescription := "Identity Provider used for drift detection testing - DO NOT MODIFY"
	expectedIssuer := "https://accounts.google.com"
	expectedJwksUri := "https://www.googleapis.com/oauth2/v3/certs"
	// ========================================

	idpDataSourceLabel := "drift_test_idp"
	idpDataSourceName := fmt.Sprintf("data.confluent_identity_provider.%s", idpDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// Read the pre-existing identity provider via data source and validate all attributes
				// Using a data source (not resource import) ensures the provider is never destroyed
				Config: testAccCheckIdentityProviderDriftConfig(endpoint, apiKey, apiSecret, idpDataSourceLabel, identityProviderId),
				Check: resource.ComposeTestCheckFunc(
					// Verify core identifiers
					resource.TestCheckResourceAttr(idpDataSourceName, "id", identityProviderId),

					// Verify configurable attributes - any mismatch indicates drift
					resource.TestCheckResourceAttr(idpDataSourceName, "display_name", expectedDisplayName),
					resource.TestCheckResourceAttr(idpDataSourceName, "description", expectedDescription),
					resource.TestCheckResourceAttr(idpDataSourceName, "issuer", expectedIssuer),
					resource.TestCheckResourceAttr(idpDataSourceName, "jwks_uri", expectedJwksUri),
				),
			},
		},
	})
}

// testAccCheckIdentityProviderDriftConfig generates Terraform configuration that reads
// a pre-existing identity provider via data source. The data source approach is intentional:
// - Data sources are read-only and never trigger resource destruction
// - Validates the provider's Read function accuracy against the Confluent Cloud API
func testAccCheckIdentityProviderDriftConfig(endpoint, apiKey, apiSecret, idpDataSourceLabel, identityProviderId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	data "confluent_identity_provider" "%s" {
		id = "%s"
	}
	`, endpoint, apiKey, apiSecret, idpDataSourceLabel, identityProviderId)
}
