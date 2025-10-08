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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceIdentityProviderLive(t *testing.T) {
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
	idpResourceLabel := "test_idp_for_ds"
	idpDisplayName := fmt.Sprintf("tf-live-idp-ds-%d", randomSuffix)

	idpResourceName := fmt.Sprintf("confluent_identity_provider.%s", idpResourceLabel)
	dataSourceIdLookupName := "data.confluent_identity_provider.by_id"
	dataSourceNameLookupName := "data.confluent_identity_provider.by_name"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIdentityProviderLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName),
				Check: resource.ComposeTestCheckFunc(
					// Ensure the underlying resource exists
					testAccCheckIdentityProviderLiveExists(idpResourceName),
					// Check that the data source lookup by ID returns the correct attributes
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "display_name", idpResourceName, "display_name"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "description", idpResourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "issuer", idpResourceName, "issuer"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "jwks_uri", idpResourceName, "jwks_uri"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "identity_claim", idpResourceName, "identity_claim"),

					// Check that the data source lookup by display_name returns the correct attributes
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "id", idpResourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "description", idpResourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "issuer", idpResourceName, "issuer"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "jwks_uri", idpResourceName, "jwks_uri"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "identity_claim", idpResourceName, "identity_claim"),
				),
			},
		},
	})
}

func testAccCheckDataSourceIdentityProviderLiveConfig(endpoint, apiKey, apiSecret, resourceLabel, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description  = "Identity Provider for Data Source live test"
		issuer       = "https://login.microsoftonline.com/common/v2.0"
		jwks_uri     = "https://login.microsoftonline.com/common/discovery/v2.0/keys"
	}

	data "confluent_identity_provider" "by_id" {
		id = confluent_identity_provider.%s.id
	}

	data "confluent_identity_provider" "by_name" {
		display_name = confluent_identity_provider.%s.display_name
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, displayName, resourceLabel, resourceLabel)
}
