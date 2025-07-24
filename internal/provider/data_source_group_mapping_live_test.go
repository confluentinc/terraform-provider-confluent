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

func TestAccGroupMappingDataSourceLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Skip test if required environment variables are not set
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	groupMappingDisplayName := fmt.Sprintf("tf-live-group-mapping-ds-%d", randomSuffix)
	groupMappingResourceLabel := "test_live_group_mapping_resource"
	groupMappingDataSourceLabel := "test_live_group_mapping_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGroupMappingDataSourceLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDataSourceLabel, groupMappingDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_group_mapping.%s", groupMappingDataSourceLabel), "id",
						fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_group_mapping.%s", groupMappingDataSourceLabel), "display_name",
						fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "display_name",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_group_mapping.%s", groupMappingDataSourceLabel), "filter",
						fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "filter",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_group_mapping.%s", groupMappingDataSourceLabel), "description",
						fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "description",
					),
				),
			},
		},
	})
}

func testAccCheckGroupMappingDataSourceLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDataSourceLabel, groupMappingDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_group_mapping" "%s" {
		display_name = "%s"
		filter       = "\"engineering\" in groups"
		description  = "Live test group mapping for engineering team"
	}

	data "confluent_group_mapping" "%s" {
		id = confluent_group_mapping.%s.id
	}
	`, endpoint, apiKey, apiSecret, groupMappingResourceLabel, groupMappingDisplayName, groupMappingDataSourceLabel, groupMappingResourceLabel)
} 