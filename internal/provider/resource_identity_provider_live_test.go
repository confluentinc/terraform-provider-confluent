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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccIdentityProviderLive(t *testing.T) {
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
	identityProviderDisplayName := fmt.Sprintf("tf-live-idp-%d", randomSuffix)
	identityProviderResourceLabel := "test_live_identity_provider"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIdentityProviderLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIdentityProviderLiveConfig(endpoint, identityProviderResourceLabel, identityProviderDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderLiveExists(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel), "display_name", identityProviderDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel), "description", "Test identity provider for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel), "issuer", "https://login.microsoftonline.com/common/discovery/v2.0"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel), "jwks_uri", "https://login.microsoftonline.com/common/discovery/v2.0/keys"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
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
		req := c.oidcClient.IdentityProvidersIamV2Api.GetIamV2IdentityProvider(c.oidcApiContext(context.Background()), deletedIdentityProviderId)
		deletedIdentityProvider, response, err := req.Execute()
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(response)
		if isResourceNotFound {
			return nil
		} else if err == nil && deletedIdentityProvider.Id == nil {
			// Otherwise return the error
			if *deletedIdentityProvider.Id == rs.Primary.ID {
				return fmt.Errorf("identity provider (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

// Configuration functions for each test case
func testAccCheckIdentityProviderLiveConfig(endpoint, identityProviderResourceLabel, identityProviderDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint 			= "%s"
		cloud_api_key 		= "%s"
		cloud_api_secret 	= "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description = "Test identity provider for live testing"
		issuer = "https://login.microsoftonline.com/common/discovery/v2.0"
		jwks_uri = "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	}
	`, endpoint, apiKey, apiSecret, identityProviderResourceLabel, identityProviderDisplayName)
}