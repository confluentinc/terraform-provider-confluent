//go:build live_test && (all || rbac)

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
	"context"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	// Using Okta's OIDC configuration for the update step to test changing providers
	updatedIssuer  = "https://example.okta.com/oauth2/default"
	updatedJwksUri = "https://example.okta.com/oauth2/default/v1/keys"
)

func TestAccIdentityProviderLive(t *testing.T) {
	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
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
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	identityProviderResourceLabel := "test_live_identity_provider"
	resourceName := fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel)

	// Initial resource attributes
	initialDisplayName := fmt.Sprintf("tf-live-idp-%d", randomSuffix)
	initialDescription := "Test IdP (Initial)"
	initialIssuer := "https://login.microsoftonline.com/common/v2.0"
	initialJwksUri := "https://login.microsoftonline.com/common/discovery/v2.0/keys"

	// Updated resource attributes
	updatedDisplayName := fmt.Sprintf("tf-live-idp-updated-%d", randomSuffix)
	updatedDescription := "Test IdP (Updated with Okta OIDC)"
	updatedIdentityClaim := "claims.email"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIdentityProviderLiveDestroy,
		Steps: []resource.TestStep{
			// Step 1: Test Create
			{
				Config: testAccCheckIdentityProviderLiveConfig(endpoint, identityProviderResourceLabel, initialDisplayName, initialDescription, initialIssuer, initialJwksUri, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderLiveExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "display_name", initialDisplayName),
					resource.TestCheckResourceAttr(resourceName, "description", initialDescription),
					resource.TestCheckResourceAttr(resourceName, "issuer", initialIssuer),
					resource.TestCheckResourceAttr(resourceName, "jwks_uri", initialJwksUri),
					resource.TestCheckResourceAttr(resourceName, "identity_claim", "claims.sub"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Step 2: Test Update all attributes
			{
				Config: testAccCheckIdentityProviderLiveConfigWithUpdate(endpoint, identityProviderResourceLabel, updatedDisplayName, updatedDescription, updatedIssuer, updatedJwksUri, updatedIdentityClaim, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderLiveExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "display_name", updatedDisplayName),
					resource.TestCheckResourceAttr(resourceName, "description", updatedDescription),
					resource.TestCheckResourceAttr(resourceName, "issuer", updatedIssuer),
					resource.TestCheckResourceAttr(resourceName, "jwks_uri", updatedJwksUri),
					resource.TestCheckResourceAttr(resourceName, "identity_claim", updatedIdentityClaim),
				),
			},
			// Step 3: Test Import (after update)
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Step 4: Test API Error Handling with invalid input
			{
				Config: testAccCheckIdentityProviderLiveConfig(
					endpoint,
					identityProviderResourceLabel,
					"Bad IdP",
					"This should fail",
					"https://login.microsoftonline.com/common/v2.0",
					"https://example.com/this/uri/is/invalid", // Invalid JWKS URI
					apiKey,
					apiSecret,
				),
				ExpectError: regexp.MustCompile("(Unable to verify Jwks URI|jwks uri.*may be invalid)"),
			},
		},
	})
}

// Helper function for checking if the identity provider exists
func testAccCheckIdentityProviderLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]

		if !ok {
			return fmt.Errorf("%s identity provider has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s identity provider", resourceName)
		}

		c := testAccProvider.Meta().(*Client)
		_, resp, err := c.oidcClient.IdentityProvidersIamV2Api.GetIamV2IdentityProvider(c.oidcApiContext(context.Background()), rs.Primary.ID).Execute()

		if err != nil {
			return fmt.Errorf("identity provider (%s) was not found: %s", rs.Primary.ID, createDescriptiveError(err, resp))
		}

		return nil
	}
}

// Helper function for checking if the identity provider has been destroyed
func testAccCheckIdentityProviderLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verify that the identity provider has been destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_identity_provider" {
			continue
		}
		deletedIdentityProviderId := rs.Primary.ID
		_, response, err := c.oidcClient.IdentityProvidersIamV2Api.GetIamV2IdentityProvider(c.oidcApiContext(context.Background()), deletedIdentityProviderId).Execute()

		if err != nil {
			if isNonKafkaRestApiResourceNotFound(response) {
				// Resource is gone as expected
				return nil
			}
			// An unexpected error occurred
			return fmt.Errorf("unexpected error checking for deleted identity provider %q: %s", rs.Primary.ID, err)
		}

		// If err is nil, the resource still exists
		return fmt.Errorf("identity provider (%q) still exists", rs.Primary.ID)
	}
	return nil
}

// Configuration function for the initial create step
func testAccCheckIdentityProviderLiveConfig(endpoint, resourceLabel, displayName, description, issuer, jwksUri, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description  = "%s"
		issuer       = "%s"
		jwks_uri     = "%s"
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, displayName, description, issuer, jwksUri)
}

// Configuration function for the update step, including the optional identity_claim
func testAccCheckIdentityProviderLiveConfigWithUpdate(endpoint, resourceLabel, displayName, description, issuer, jwksUri, identityClaim, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name   = "%s"
		description    = "%s"
		issuer         = "%s"
		jwks_uri       = "%s"
		identity_claim = "%s"
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, displayName, description, issuer, jwksUri, identityClaim)
}
