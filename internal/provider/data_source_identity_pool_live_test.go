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

func TestAccDataSourceIdentityPoolLive(t *testing.T) {
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
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources
	randomSuffix := rand.Intn(100000)
	idpResourceLabel := "test_idp_for_ds"
	poolResourceLabel := "test_pool_for_ds"
	idpDisplayName := fmt.Sprintf("tf-live-idp-ds-%d", randomSuffix)
	poolDisplayName := fmt.Sprintf("tf-live-pool-ds-%d", randomSuffix)

	poolResourceName := fmt.Sprintf("confluent_identity_pool.%s", poolResourceLabel)
	dataSourceIdLookupName := "data.confluent_identity_pool.by_id"
	dataSourceNameLookupName := "data.confluent_identity_pool.by_name"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, poolDisplayName),
				Check: resource.ComposeTestCheckFunc(
					// Ensure the underlying resource exists
					testAccCheckIdentityPoolLiveExists(poolResourceName),
					// Check that the data source lookup by ID returns the correct attributes
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "display_name", poolResourceName, "display_name"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "description", poolResourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "identity_claim", poolResourceName, "identity_claim"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "filter", poolResourceName, "filter"),
					resource.TestCheckResourceAttrPair(dataSourceIdLookupName, "identity_provider.0.id", poolResourceName, "identity_provider.0.id"),

					// Check that the data source lookup by display_name returns the correct attributes
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "id", poolResourceName, "id"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "description", poolResourceName, "description"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "identity_claim", poolResourceName, "identity_claim"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "filter", poolResourceName, "filter"),
					resource.TestCheckResourceAttrPair(dataSourceNameLookupName, "identity_provider.0.id", poolResourceName, "identity_provider.0.id"),
				),
			},
		},
	})
}

func testAccCheckDataSourceIdentityPoolLiveConfig(endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, poolDisplayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description  = "Identity Provider for Identity Pool Data Source live test"
		issuer       = "https://accounts.google.com"
		jwks_uri     = "https://www.googleapis.com/oauth2/v3/certs"
	}

	resource "confluent_identity_pool" "%s" {
		identity_provider {
			id = confluent_identity_provider.%s.id
		}
		display_name   = "%s"
		description    = "Identity Pool for Data Source live test"
		identity_claim = "claims.sub"
		filter         = "claims.aud == \"tf-acc-test\""
	}

	data "confluent_identity_pool" "by_id" {
		identity_provider {
			id = confluent_identity_provider.%s.id
		}
		id = confluent_identity_pool.%s.id
	}

	data "confluent_identity_pool" "by_name" {
		identity_provider {
			id = confluent_identity_provider.%s.id
		}
		display_name = confluent_identity_pool.%s.display_name
	}
	`, endpoint, apiKey, apiSecret, idpResourceLabel, idpDisplayName, poolResourceLabel, idpResourceLabel, poolDisplayName, idpResourceLabel, poolResourceLabel, idpResourceLabel, poolResourceLabel)
}