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
)

func TestAccServiceAccountDataSourceLive(t *testing.T) {
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
	serviceAccountDisplayName := fmt.Sprintf("tf-live-sa-ds-%d", randomSuffix)
	serviceAccountResourceLabel := "test_live_service_account_resource"
	serviceAccountDataSourceLabel := "test_live_service_account_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckServiceAccountDataSourceLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDataSourceLabel, serviceAccountDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the resource was created
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "display_name", serviceAccountDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "description", "A test service account for live testing data source"),
					
					// Check the data source can find it
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_service_account.%s", serviceAccountDataSourceLabel), "id",
						fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_service_account.%s", serviceAccountDataSourceLabel), "display_name",
						fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "display_name",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_service_account.%s", serviceAccountDataSourceLabel), "description",
						fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "description",
					),
				),
			},
		},
	})
}

func testAccCheckServiceAccountDataSourceLiveConfig(endpoint, serviceAccountResourceLabel, serviceAccountDataSourceLabel, serviceAccountDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description  = "A test service account for live testing data source"
	}

	data "confluent_service_account" "%s" {
		id = confluent_service_account.%s.id
	}
	`, endpoint, apiKey, apiSecret, serviceAccountResourceLabel, serviceAccountDisplayName, serviceAccountDataSourceLabel, serviceAccountResourceLabel)
} 