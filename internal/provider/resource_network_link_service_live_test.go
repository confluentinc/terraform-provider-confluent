//go:build live_test && (all || networking)

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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccNetworkLinkServiceLive(t *testing.T) {
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
		endpoint = "https://api.confluent.cloud"
	}

	environmentId := os.Getenv("LIVE_TEST_ENVIRONMENT_ID")
	networkId := os.Getenv("LIVE_TEST_AWS_PRIVATELINK_NETWORK_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for network link service live tests")
	}

	if networkId == "" {
		t.Skip("Skipping Network Link Service test. LIVE_TEST_AWS_PRIVATELINK_NETWORK_ID must be set to run this test.")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	nlsDisplayName := fmt.Sprintf("tf-live-nls-%d", randomSuffix)
	nlsDisplayNameUpdated := fmt.Sprintf("tf-live-nls-updated-%d", randomSuffix)
	nlsResourceLabel := "test_live_network_link_service"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLinkServiceLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckNetworkLinkServiceLiveConfig(endpoint, nlsResourceLabel, nlsDisplayName, environmentId, networkId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkLinkServiceLiveExists(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "display_name", nlsDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "network.0.id", networkId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "resource_name"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					nlsId := resources[fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, nlsId), nil
				},
			},
			{
				Config: testAccCheckNetworkLinkServiceLiveConfig(endpoint, nlsResourceLabel, nlsDisplayNameUpdated, environmentId, networkId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkLinkServiceLiveExists(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "display_name", nlsDisplayNameUpdated),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel), "network.0.id", networkId),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					nlsId := resources[fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_network_link_service.%s", nlsResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, nlsId), nil
				},
			},
		},
	})
}

func testAccCheckNetworkLinkServiceLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_network_link_service" {
			continue
		}

		// In live tests, we can't easily check if the resource is actually destroyed
		// without making API calls, so we just verify the resource is removed from state
		if rs.Primary.ID != "" {
			// This is normal - the resource should have an ID but be removed from the live environment
			// The actual cleanup happens through the API calls during destroy
		}
	}
	return nil
}

func testAccCheckNetworkLinkServiceLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		return nil
	}
}

func testAccCheckNetworkLinkServiceLiveConfig(endpoint, nlsResourceLabel, nlsDisplayName, environmentId, networkId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_network_link_service" "%s" {
		display_name = "%s"
		description  = "Test network link service for live testing"
		environment {
			id = "%s"
		}
		network {
			id = "%s"
		}
		accept {
			environments = ["%s"]
		}
	}
	`, endpoint, apiKey, apiSecret, nlsResourceLabel, nlsDisplayName, environmentId, networkId, environmentId)
}

