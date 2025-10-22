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
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccIPGroupLive(t *testing.T) {
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
	ipGroupName := fmt.Sprintf("tf-live-ip-group-%d", randomSuffix)
	ipGroupUpdatedName := fmt.Sprintf("tf-live-ip-group-updated-%d", randomSuffix)
	ipGroupResourceLabel := "test_live_ip_group"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIPGroupLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIPGroupLiveConfig(endpoint, ipGroupResourceLabel, ipGroupName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupLiveExists(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "group_name", ipGroupName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.#", "2"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.*", "192.168.0.0/24"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.*", "10.0.0.0/16"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckIPGroupUpdateLiveConfig(endpoint, ipGroupResourceLabel, ipGroupUpdatedName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupLiveExists(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "group_name", ipGroupUpdatedName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.#", "3"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.*", "172.16.0.0/12"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.*", "10.1.0.0/16"),
					resource.TestCheckTypeSetElemAttr(fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel), "cidr_blocks.*", "192.168.100.0/24"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_ip_group.%s", ipGroupResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckIPGroupLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each IP Group is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_ip_group" {
			continue
		}
		deletedIPGroupId := rs.Primary.ID
		req := c.iamIPClient.IPGroupsIamV2Api.GetIamV2IpGroup(c.iamIPApiContext(context.Background()), deletedIPGroupId)
		deletedIPGroup, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// If the error is equivalent to http.StatusNotFound, the IP Group is destroyed.
			return nil
		} else if err == nil && deletedIPGroup.Id != nil {
			// Otherwise return the error
			if *deletedIPGroup.Id == rs.Primary.ID {
				return fmt.Errorf("IP Group (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckIPGroupLiveConfig(endpoint, resourceLabel, groupName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"192.168.0.0/24",
			"10.0.0.0/16"
		]
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, groupName)
}

func testAccCheckIPGroupUpdateLiveConfig(endpoint, resourceLabel, groupName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			"172.16.0.0/12",
			"10.1.0.0/16",
			"192.168.100.0/24"
		]
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, groupName)
}

func testAccCheckIPGroupLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s IP Group has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s IP Group", n)
		}

		return nil
	}
}

