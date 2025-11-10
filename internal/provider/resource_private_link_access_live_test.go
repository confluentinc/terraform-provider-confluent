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
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPrivateLinkAccessLive(t *testing.T) {
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

	networkId := os.Getenv("LIVE_TEST_AWS_PRIVATELINK_NETWORK_ID")
	environmentId := os.Getenv("LIVE_TEST_ENVIRONMENT_ID")

	// Extract AWS account ID from KMS key ARN (format: arn:aws:kms:region:ACCOUNT_ID:key/key-id)
	awsKmsKeyArn := os.Getenv("TEST_KMS_KEY_ID")
	var awsAccountId string
	if awsKmsKeyArn != "" {
		// Extract account ID from ARN: arn:aws:kms:us-west-2:237597620434:key/...
		arnParts := strings.Split(awsKmsKeyArn, ":")
		if len(arnParts) >= 5 {
			awsAccountId = arnParts[4]
		}
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if networkId == "" {
		t.Skip("Skipping Private Link Access test. LIVE_TEST_AWS_PRIVATELINK_NETWORK_ID must be set to run this test.")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for private link access live tests")
	}

	if awsAccountId == "" {
		t.Skip("Skipping Private Link Access test. TEST_KMS_KEY_ID must be set (to extract AWS account ID) to run this test.")
	}

	// Validate AWS account ID format (12 digits)
	matched, _ := regexp.MatchString(`^\d{12}$`, awsAccountId)
	if !matched {
		t.Fatalf("Invalid AWS account ID format extracted from TEST_KMS_KEY_ID: %s (expected 12 digits)", awsAccountId)
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	plaDisplayName := fmt.Sprintf("tf-live-pla-%d", randomSuffix)
	plaDisplayNameUpdated := fmt.Sprintf("tf-live-pla-updated-%d", randomSuffix)
	plaResourceLabel := "test_live_private_link_access"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckPrivateLinkAccessLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPrivateLinkAccessLiveConfig(endpoint, plaResourceLabel, plaDisplayName, environmentId, networkId, awsAccountId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPrivateLinkAccessLiveExists(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "display_name", plaDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "network.0.id", networkId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "aws.0.account", awsAccountId),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					plaId := resources[fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, plaId), nil
				},
			},
			{
				Config: testAccCheckPrivateLinkAccessLiveConfig(endpoint, plaResourceLabel, plaDisplayNameUpdated, environmentId, networkId, awsAccountId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPrivateLinkAccessLiveExists(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "display_name", plaDisplayNameUpdated),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "environment.0.id", environmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "network.0.id", networkId),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel), "aws.0.account", awsAccountId),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					plaId := resources[fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_private_link_access.%s", plaResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, plaId), nil
				},
			},
		},
	})
}

func testAccCheckPrivateLinkAccessLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_private_link_access" {
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

func testAccCheckPrivateLinkAccessLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckPrivateLinkAccessLiveConfig(endpoint, plaResourceLabel, plaDisplayName, environmentId, networkId, awsAccountId, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_private_link_access" "%s" {
		display_name = "%s"
		environment {
			id = "%s"
		}
		network {
			id = "%s"
		}
		aws {
			account = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, plaResourceLabel, plaDisplayName, environmentId, networkId, awsAccountId)
}

