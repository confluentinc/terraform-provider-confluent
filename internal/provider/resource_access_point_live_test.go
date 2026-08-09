//go:build live_test && (all || networking)

// Copyright 2024 Confluent Inc. All Rights Reserved.
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

// Exercises the GCP egress Private Service Connect variant. ALL_GOOGLE_APIS is a
// Google-managed global target, so it needs no customer-side PSC service standing up
// first -- unlike the AWS/Azure egress variants, which each need a real endpoint
// service name from outside Confluent.
//
// The gateway it attaches to is provisioned inline rather than pre-existing: creating
// a confluent_network with connection_types = ["PRIVATELINK"] auto-provisions an
// associated gateway, exposed as the network's own computed gateway[0].id --
// confluent_access_point's and confluent_dns_record's own published docs reference it
// exactly this way (confluent_network.main.gateway[0].id). So the whole chain is
// created and destroyed by this test; nothing needs pre-provisioning beyond API
// credentials.
func TestAccAccessPointGcpEgressLive(t *testing.T) {
	t.Parallel()

	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	randomSuffix := rand.Intn(100000)
	environmentResourceLabel := "test_live_env"
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", randomSuffix)
	networkResourceLabel := "test_live_network"
	networkDisplayName := fmt.Sprintf("tf-live-network-%d", randomSuffix)
	accessPointResourceLabel := "test_live_access_point"
	accessPointDisplayName := fmt.Sprintf("tf-live-access-point-%d", randomSuffix)
	accessPointDisplayNameUpdated := fmt.Sprintf("tf-live-access-point-updated-%d", randomSuffix)
	fullAccessPointLabel := fmt.Sprintf("confluent_access_point.%s", accessPointResourceLabel)
	fullNetworkLabel := fmt.Sprintf("confluent_network.%s", networkResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAccessPointLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAccessPointGcpEgressLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, accessPointResourceLabel, accessPointDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessPointLiveExists(fullAccessPointLabel),
					resource.TestCheckResourceAttr(fullAccessPointLabel, "display_name", accessPointDisplayName),
					resource.TestCheckResourceAttrSet(fullAccessPointLabel, "id"),
					resource.TestCheckResourceAttrSet(fullAccessPointLabel, "environment.0.id"),
					resource.TestCheckResourceAttrPair(fullAccessPointLabel, "gateway.0.id", fullNetworkLabel, "gateway.0.id"),
					resource.TestCheckResourceAttr(fullAccessPointLabel, "gcp_egress_private_service_connect_endpoint.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_target", "ALL_GOOGLE_APIS"),
					resource.TestCheckResourceAttrSet(fullAccessPointLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_connection_id"),
					resource.TestCheckResourceAttrSet(fullAccessPointLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_ip_address"),
				),
			},
			{
				ResourceName:      fullAccessPointLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					accessPointId := resources[fullAccessPointLabel].Primary.ID
					environmentId := resources[fullAccessPointLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + accessPointId, nil
				},
			},
			{
				// display_name is the only field this variant can update without
				// forcing recreation -- the gcp_egress block and the gateway it
				// attaches to are both ForceNew.
				Config: testAccCheckAccessPointGcpEgressLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, accessPointResourceLabel, accessPointDisplayNameUpdated, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccessPointLiveExists(fullAccessPointLabel),
					resource.TestCheckResourceAttr(fullAccessPointLabel, "display_name", accessPointDisplayNameUpdated),
				),
			},
		},
	})
}

func testAccCheckAccessPointGcpEgressLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, accessPointResourceLabel, accessPointDisplayName, apiKey, apiSecret string) string {
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
		cloud            = "GCP"
		region           = "us-central1"
		connection_types = ["PRIVATELINK"]
		environment {
			id = confluent_environment.%s.id
		}
	}

	resource "confluent_access_point" "%s" {
		display_name = "%s"
		environment {
			id = confluent_environment.%s.id
		}
		gateway {
			id = confluent_network.%s.gateway[0].id
		}
		gcp_egress_private_service_connect_endpoint {
			private_service_connect_endpoint_target = "ALL_GOOGLE_APIS"
		}
	}
	`, endpoint, apiKey, apiSecret,
		environmentResourceLabel, environmentDisplayName,
		networkResourceLabel, networkDisplayName, environmentResourceLabel,
		accessPointResourceLabel, accessPointDisplayName, environmentResourceLabel, networkResourceLabel)
}

func testAccCheckAccessPointLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckAccessPointLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_access_point" {
			continue
		}
		deletedAccessPointId := rs.Primary.ID
		environmentId := rs.Primary.Attributes["environment.0.id"]
		req := c.networkingAccessPointV1Client.AccessPointsNetworkingV1Api.GetNetworkingV1AccessPoint(c.networkingAccessPointV1ApiContext(context.Background()), deletedAccessPointId).Environment(environmentId)
		deletedAccessPoint, response, err := req.Execute()
		if response != nil && isNonKafkaRestApiResourceNotFound(response) {
			return nil
		} else if err == nil && deletedAccessPoint.Id != nil {
			if *deletedAccessPoint.Id == rs.Primary.ID {
				return fmt.Errorf("access point (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
