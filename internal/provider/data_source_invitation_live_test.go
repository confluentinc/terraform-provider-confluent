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

func TestAccInvitationDataSourceLive(t *testing.T) {
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
	invitationEmail := fmt.Sprintf("tf-live-invitation-ds-%d@confluent.io", randomSuffix)
	invitationResourceLabel := "test_live_invitation_resource"
	invitationDataSourceLabel := "test_live_invitation_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckInvitationDataSourceLiveConfig(endpoint, invitationResourceLabel, invitationDataSourceLabel, invitationEmail, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the invitation was created
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "email", invitationEmail),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "auth_type", "AUTH_TYPE_LOCAL"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "status"),
					
					// Check the data source can find it
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_invitation.%s", invitationDataSourceLabel), "id",
						fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "id",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_invitation.%s", invitationDataSourceLabel), "email",
						fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "email",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_invitation.%s", invitationDataSourceLabel), "auth_type",
						fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "auth_type",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_invitation.%s", invitationDataSourceLabel), "status",
						fmt.Sprintf("confluent_invitation.%s", invitationResourceLabel), "status",
					),
				),
			},
		},
	})
}

func testAccCheckInvitationDataSourceLiveConfig(endpoint, invitationResourceLabel, invitationDataSourceLabel, invitationEmail, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_invitation" "%s" {
		email     = "%s"
		auth_type = "AUTH_TYPE_LOCAL"
	}

	data "confluent_invitation" "%s" {
		id = confluent_invitation.%s.id
	}
	`, endpoint, apiKey, apiSecret, invitationResourceLabel, invitationEmail, invitationDataSourceLabel, invitationResourceLabel)
} 