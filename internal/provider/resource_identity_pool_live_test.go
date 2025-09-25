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

func TestAccIdentityPoolLive(t *testing.T) {
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

	// Generate unique names for test resources
	randomSuffix := rand.Intn(100000)
	idpResourceLabel := "test_idp_for_pool"
	poolResourceLabel := "test_identity_pool"

	idpResourceName := fmt.Sprintf("confluent_identity_provider.%s", idpResourceLabel)
	poolResourceName := fmt.Sprintf("confluent_identity_pool.%s", poolResourceLabel)

	// Initial Identity Provider attributes
	idpDisplayName := fmt.Sprintf("tf-live-idp-%d", randomSuffix)

	// Initial Identity Pool attributes
	poolDisplayName := fmt.Sprintf("tf-live-pool-%d", randomSuffix)
	poolDescription := "Test Identity Pool (Initial)"
	poolIdentityClaim := "claims.sub"
	poolFilter := `claims.email.endsWith("@example.com")`

	// Updated Identity Pool attributes
	updatedPoolDisplayName := fmt.Sprintf("tf-live-pool-updated-%d", randomSuffix)
	updatedPoolDescription := "Test Identity Pool (Updated)"
	updatedPoolIdentityClaim := "claims.aud"
	updatedPoolFilter := `claims.aud == "confluent-test"`

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIdentityPoolLiveDestroy,
		Steps: []resource.TestStep{
			// Step 1: Test Create
			{
				Config: testAccCheckIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, poolDisplayName, poolDescription, poolIdentityClaim, poolFilter),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderLiveExists(idpResourceName),
					testAccCheckIdentityPoolLiveExists(poolResourceName),
					resource.TestCheckResourceAttr(poolResourceName, "display_name", poolDisplayName),
					resource.TestCheckResourceAttr(poolResourceName, "description", poolDescription),
					resource.TestCheckResourceAttr(poolResourceName, "identity_claim", poolIdentityClaim),
					resource.TestCheckResourceAttr(poolResourceName, "filter", poolFilter),
					resource.TestCheckResourceAttrPair(poolResourceName, "identity_provider.0.id", idpResourceName, "id"),
				),
			},
			// Step 2: Test Update
			{
				Config: testAccCheckIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, updatedPoolDisplayName, updatedPoolDescription, updatedPoolIdentityClaim, updatedPoolFilter),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolLiveExists(poolResourceName),
					resource.TestCheckResourceAttr(poolResourceName, "display_name", updatedPoolDisplayName),
					resource.TestCheckResourceAttr(poolResourceName, "description", updatedPoolDescription),
					resource.TestCheckResourceAttr(poolResourceName, "identity_claim", updatedPoolIdentityClaim),
					resource.TestCheckResourceAttr(poolResourceName, "filter", updatedPoolFilter),
				),
			},
			// Step 3: Test Import
			{
				ResourceName:      poolResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					pool, ok := resources[poolResourceName]
					if !ok {
						return "", fmt.Errorf("resource %s not found in state", poolResourceName)
					}
					poolID := pool.Primary.ID
					providerID := pool.Primary.Attributes["identity_provider.0.id"]
					return fmt.Sprintf("%s/%s", providerID, poolID), nil
				},
			},
			// Step 4: Test API Error Handling with invalid filter
			{
				Config:      testAccCheckIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, poolDisplayName, poolDescription, poolIdentityClaim, "this-is-not-a-valid-filter"),
				ExpectError: regexp.MustCompile("Invalid syntax in policy expression"),
			},
		},
	})
}

func testAccCheckIdentityPoolLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s identity pool has not been found", resourceName)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s identity pool", resourceName)
		}
		return nil // Existence is verified by the Read call in the lifecycle
	}
}

func testAccCheckIdentityPoolLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_identity_pool" {
			continue
		}
		identityProviderId := rs.Primary.Attributes["identity_provider.0.id"]
		identityPoolId := rs.Primary.ID

		_, response, err := c.oidcClient.IdentityPoolsIamV2Api.GetIamV2IdentityPool(c.oidcApiContext(context.Background()), identityProviderId, identityPoolId).Execute()
		if err != nil {
			if isNonKafkaRestApiResourceNotFound(response) {
				// Resource is gone as expected
				continue
			}
			return fmt.Errorf("unexpected error checking for deleted identity pool %q: %s", rs.Primary.ID, err)
		}
		return fmt.Errorf("identity pool (%q) still exists", rs.Primary.ID)
	}
	return nil
}

func testAccCheckIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, poolDisplayName, poolDescription, poolIdentityClaim, poolFilter string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description  = "Identity Provider for Identity Pool live test"
		issuer       = "https://accounts.google.com"
		jwks_uri     = "https://www.googleapis.com/oauth2/v3/certs"
	}

	resource "confluent_identity_pool" "%s" {
		identity_provider {
			id = confluent_identity_provider.%s.id
		}
		display_name   = "%s"
		description    = "%s"
		identity_claim = "%s"
		filter         = %q
	}
	`, endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, idpResourceLabel, poolDisplayName, poolDescription, poolIdentityClaim, poolFilter)
}