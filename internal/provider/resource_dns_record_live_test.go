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

// A DNS record's only oneOf variant is private_link_access_point, a reference to an
// existing access point by ID -- there is no per-cloud config here the way access_point
// has. So this creates a GCP egress access point inline (the ALL_GOOGLE_APIS variant
// needs no customer-side PSC service, same reasoning as
// TestAccAccessPointGcpEgressLive) and points the DNS record at it. Both the access
// point and the DNS record attach to the same externally-provisioned gateway.
func TestAccDnsRecordGcpLive(t *testing.T) {
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

	gatewayId := os.Getenv("CONFLUENT_CLOUD_GCP_GATEWAY_ID")
	if gatewayId == "" {
		t.Fatal("CONFLUENT_CLOUD_GCP_GATEWAY_ID must be set: a pre-provisioned GCP gateway for both the access point and the DNS record to attach to")
	}

	randomSuffix := rand.Intn(100000)
	environmentResourceLabel := "test_live_env"
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", randomSuffix)
	accessPointResourceLabel := "test_live_access_point"
	accessPointDisplayName := fmt.Sprintf("tf-live-access-point-%d", randomSuffix)
	dnsRecordResourceLabel := "test_live_dns_record"
	dnsRecordDomain := fmt.Sprintf("tf-live-dns-record-%d.example.com", randomSuffix)
	dnsRecordDisplayName := fmt.Sprintf("tf-live-dns-record-%d", randomSuffix)
	dnsRecordDisplayNameUpdated := fmt.Sprintf("tf-live-dns-record-updated-%d", randomSuffix)
	fullDnsRecordLabel := fmt.Sprintf("confluent_dns_record.%s", dnsRecordResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckDnsRecordLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDnsRecordGcpLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, accessPointResourceLabel, accessPointDisplayName, dnsRecordResourceLabel, dnsRecordDomain, dnsRecordDisplayName, gatewayId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordLiveExists(fullDnsRecordLabel),
					resource.TestCheckResourceAttr(fullDnsRecordLabel, "domain", dnsRecordDomain),
					resource.TestCheckResourceAttr(fullDnsRecordLabel, "display_name", dnsRecordDisplayName),
					resource.TestCheckResourceAttrSet(fullDnsRecordLabel, "id"),
					resource.TestCheckResourceAttrSet(fullDnsRecordLabel, "environment.0.id"),
					resource.TestCheckResourceAttr(fullDnsRecordLabel, "gateway.0.id", gatewayId),
					resource.TestCheckResourceAttrPair(fullDnsRecordLabel, "private_link_access_point.0.id", fmt.Sprintf("confluent_access_point.%s", accessPointResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fullDnsRecordLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					dnsRecordId := resources[fullDnsRecordLabel].Primary.ID
					environmentId := resources[fullDnsRecordLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + dnsRecordId, nil
				},
			},
			{
				// display_name is a real PATCH, unlike access_point's update test:
				// domain, gateway and private_link_access_point are all ForceNew, but
				// display_name is not.
				Config: testAccCheckDnsRecordGcpLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, accessPointResourceLabel, accessPointDisplayName, dnsRecordResourceLabel, dnsRecordDomain, dnsRecordDisplayNameUpdated, gatewayId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDnsRecordLiveExists(fullDnsRecordLabel),
					resource.TestCheckResourceAttr(fullDnsRecordLabel, "display_name", dnsRecordDisplayNameUpdated),
				),
			},
		},
	})
}

func testAccCheckDnsRecordGcpLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, accessPointResourceLabel, accessPointDisplayName, dnsRecordResourceLabel, dnsRecordDomain, dnsRecordDisplayName, gatewayId, apiKey, apiSecret string) string {
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

	resource "confluent_access_point" "%s" {
		display_name = "%s"
		environment {
			id = confluent_environment.%s.id
		}
		gateway {
			id = "%s"
		}
		gcp_egress_private_service_connect_endpoint {
			private_service_connect_endpoint_target = "ALL_GOOGLE_APIS"
		}
	}

	resource "confluent_dns_record" "%s" {
		domain        = "%s"
		display_name  = "%s"
		environment {
			id = confluent_environment.%s.id
		}
		gateway {
			id = "%s"
		}
		private_link_access_point {
			id = confluent_access_point.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret,
		environmentResourceLabel, environmentDisplayName,
		accessPointResourceLabel, accessPointDisplayName, environmentResourceLabel, gatewayId,
		dnsRecordResourceLabel, dnsRecordDomain, dnsRecordDisplayName, environmentResourceLabel, gatewayId, accessPointResourceLabel)
}

func testAccCheckDnsRecordLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckDnsRecordLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_dns_record" {
			continue
		}
		deletedDnsRecordId := rs.Primary.ID
		environmentId := rs.Primary.Attributes["environment.0.id"]
		req := c.networkingAccessPointV1Client.DNSRecordsNetworkingV1Api.GetNetworkingV1DnsRecord(c.networkingAccessPointV1ApiContext(context.Background()), deletedDnsRecordId).Environment(environmentId)
		deletedDnsRecord, response, err := req.Execute()
		if response != nil && isNonKafkaRestApiResourceNotFound(response) {
			return nil
		} else if err == nil && deletedDnsRecord.Id != nil {
			if *deletedDnsRecord.Id == rs.Primary.ID {
				return fmt.Errorf("DNS record (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
