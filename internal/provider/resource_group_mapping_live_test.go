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
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGroupMappingLive(t *testing.T) {
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
	groupMappingDisplayName := fmt.Sprintf("tf-live-group-mapping-%d", randomSuffix)
	groupMappingResourceLabel := "test_live_group_mapping"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGroupMappingLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGroupMappingLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingLiveExists(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "display_name", groupMappingDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "filter", "\"engineering\" in groups"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "description", "Live test group mapping for engineering team"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGroupMappingUpdateLive(t *testing.T) {
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
	groupMappingDisplayName := fmt.Sprintf("tf-live-group-mapping-update-%d", randomSuffix)
	groupMappingResourceLabel := "test_live_group_mapping_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGroupMappingLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGroupMappingLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingLiveExists(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "display_name", groupMappingDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "filter", "\"engineering\" in groups"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "description", "Live test group mapping for engineering team"),
				),
			},
			{
				Config: testAccCheckGroupMappingUpdateLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingLiveExists(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "display_name", fmt.Sprintf("%s-updated", groupMappingDisplayName)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "filter", "\"devops\" in groups"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel), "description", "Updated live test group mapping for devops team"),
				),
			},
		},
	})
}

func testAccCheckGroupMappingLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each group mapping is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_group_mapping" {
			continue
		}
		deletedGroupMappingId := rs.Primary.ID
		req := c.ssoClient.GroupMappingsIamV2SsoApi.GetIamV2SsoGroupMapping(c.ssoApiContext(context.Background()), deletedGroupMappingId)
		deletedGroupMapping, response, err := req.Execute()
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(response)
		if isResourceNotFound {
			return nil
		} else if err == nil && deletedGroupMapping.Id != nil {
			// Otherwise return the error
			if *deletedGroupMapping.Id == rs.Primary.ID {
				return fmt.Errorf("group mapping (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckGroupMappingLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s group mapping has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s group mapping", n)
		}

		return nil
	}
}

func testAccCheckGroupMappingLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDisplayName, apiKey, apiSecret string) string {
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
	`, endpoint, apiKey, apiSecret, groupMappingResourceLabel, groupMappingDisplayName)
}

func testAccCheckGroupMappingUpdateLiveConfig(endpoint, groupMappingResourceLabel, groupMappingDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_group_mapping" "%s" {
		display_name = "%s-updated"
		filter       = "\"devops\" in groups"
		description  = "Updated live test group mapping for devops team"
	}
	`, endpoint, apiKey, apiSecret, groupMappingResourceLabel, groupMappingDisplayName)
} 