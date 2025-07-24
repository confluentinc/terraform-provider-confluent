//go:build live_test && (all || rbac)

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

func TestAccRoleBindingDataSourceLive(t *testing.T) {
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
	serviceAccountDisplayName := fmt.Sprintf("tf-live-sa-rb-ds-%d", randomSuffix)
	serviceAccountResourceLabel := "test_live_service_account_resource"
	roleBindingResourceLabel := "test_live_role_binding_resource"
	roleBindingDataSourceLabel := "test_live_role_binding_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRoleBindingDataSourceLiveConfig(endpoint, serviceAccountResourceLabel, roleBindingResourceLabel, roleBindingDataSourceLabel, serviceAccountDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the service account was created
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_service_account.%s", serviceAccountResourceLabel), "display_name", serviceAccountDisplayName),
					
					// Check the role binding was created
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "role_name", "MetricsViewer"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "crn_pattern", "crn://confluent.cloud/organization=424fb7bf-40c2-433f-81a5-c45942a6a539"),
					
					// Check the data source can find it
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_role_binding.%s", roleBindingDataSourceLabel), "id",
						fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_role_binding.%s", roleBindingDataSourceLabel), "role_name",
						fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "role_name",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_role_binding.%s", roleBindingDataSourceLabel), "crn_pattern",
						fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "crn_pattern",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_role_binding.%s", roleBindingDataSourceLabel), "principal",
						fmt.Sprintf("confluent_role_binding.%s", roleBindingResourceLabel), "principal",
					),
				),
			},
		},
	})
}

func testAccCheckRoleBindingDataSourceLiveConfig(endpoint, serviceAccountResourceLabel, roleBindingResourceLabel, roleBindingDataSourceLabel, serviceAccountDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_service_account" "%s" {
		display_name = "%s"
		description  = "A test service account for role binding data source live testing"
	}

	resource "confluent_role_binding" "%s" {
		principal   = "User:${confluent_service_account.%s.id}"
		role_name   = "MetricsViewer"
		crn_pattern = "crn://confluent.cloud/organization=424fb7bf-40c2-433f-81a5-c45942a6a539"
	}

	data "confluent_role_binding" "%s" {
		id = confluent_role_binding.%s.id
	}
	`, endpoint, apiKey, apiSecret, serviceAccountResourceLabel, serviceAccountDisplayName, roleBindingResourceLabel, serviceAccountResourceLabel, roleBindingDataSourceLabel, roleBindingResourceLabel)
} 