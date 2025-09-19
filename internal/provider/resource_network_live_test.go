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
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Test Private Link network for Enterprise clusters
func TestAccNetworkPrivateLinkLive(t *testing.T) {
	t.Skip()
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

	networkDisplayName := fmt.Sprintf("tf-live-network-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	networkResourceLabel := "test_live_network"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckNetworkPrivateLinkLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Core network attributes per API spec
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "display_name", networkDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "connection_types.#", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "connection_types.0", "PRIVATELINK"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "resource_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "environment.0.id"),
					// PrivateLink specific attributes per API spec
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "dns_domain"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "endpoint_suffix"),
					// AWS cloud specific attributes per API spec
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "aws.0.vpc"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "aws.0.account"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_network.%s", networkResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					networkId := resources[fmt.Sprintf("confluent_network.%s", networkResourceLabel)].Primary.ID
					environmentId := resources[fmt.Sprintf("confluent_network.%s", networkResourceLabel)].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + networkId, nil
				},
			},
		},
	})
}

// Test VPC Peering network
func TestAccNetworkVpcPeeringLive(t *testing.T) {
	t.Skip()
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

	networkDisplayName := fmt.Sprintf("tf-live-peering-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	networkResourceLabel := "test_live_peering_network"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckNetworkLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckNetworkVpcPeeringLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Core network attributes per API spec
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "display_name", networkDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "connection_types.#", "1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "connection_types.0", "PEERING"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "resource_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "environment.0.id"),
					// VPC Peering specific attributes per API spec
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "cidr", "10.10.0.0/16"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "zones.#"),
					// AWS cloud specific attributes per API spec
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "aws.0.vpc"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_network.%s", networkResourceLabel), "aws.0.account"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_network.%s", networkResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					networkId := resources[fmt.Sprintf("confluent_network.%s", networkResourceLabel)].Primary.ID
					environmentId := resources[fmt.Sprintf("confluent_network.%s", networkResourceLabel)].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + networkId, nil
				},
			},
		},
	})
}

// Configuration for Private Link network
func testAccCheckNetworkPrivateLinkLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
		stream_governance {
			package = "ESSENTIALS"
		}
	}

	resource "confluent_network" "%s" {
		display_name     = "%s"
		cloud            = "AWS"
		region           = "us-east-1"
		connection_types = ["PRIVATELINK"]
		
		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, environmentResourceLabel)
}

// Configuration for VPC Peering network
func testAccCheckNetworkVpcPeeringLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
		stream_governance {
			package = "ESSENTIALS"
		}
	}

	resource "confluent_network" "%s" {
		display_name     = "%s"
		cloud            = "AWS"
		region           = "us-east-1"
		connection_types = ["PEERING"]
		cidr             = "10.10.0.0/16"
		
		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, environmentResourceLabel)
}

// Helper function to verify network is properly cleaned up for live tests
func testAccCheckNetworkLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each network is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_network" {
			continue
		}
		deletedNetworkId := rs.Primary.ID
		// Get the actual environment ID from the state, not a hardcoded constant
		environmentId := rs.Primary.Attributes["environment.0.id"]
		req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(context.Background()), deletedNetworkId).Environment(environmentId)
		deletedNetwork, response, err := req.Execute()
		if response != nil && isNonKafkaRestApiResourceNotFound(response) {
			// networking/v1/networks/{nonExistentNetworkId/deletedNetworkID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the network is destroyed.
			return nil
		} else if err == nil && deletedNetwork.Id != nil {
			// Network still exists
			if *deletedNetwork.Id == rs.Primary.ID {
				return fmt.Errorf("network (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
