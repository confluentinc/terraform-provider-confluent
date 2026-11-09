//go:build live_test && (all || crp)

// Copyright 2026 Confluent Inc. All Rights Reserved.
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

func TestAccClientRequestPolicyLive(t *testing.T) {
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

	// The CRN of a CRP-enabled Kafka cluster the policy will be bound to.
	resourceCrn := os.Getenv("CLIENT_REQUEST_POLICY_RESOURCE_CRN")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}
	if resourceCrn == "" {
		t.Skip("Skipping live test. Set CLIENT_REQUEST_POLICY_RESOURCE_CRN to the CRN of a CRP-enabled cluster to run this test.")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	policyName := fmt.Sprintf("tf-live-crp-%d", randomSuffix)
	resourceLabel := "test_live_client_request_policy"
	fullResourceLabel := fmt.Sprintf("confluent_client_request_policy.%s", resourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClientRequestPolicyLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClientRequestPolicyLiveConfig(endpoint, resourceLabel, policyName, resourceCrn, apiKey, apiSecret, "AUDIT", "request.client.version >= '3.5.0'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClientRequestPolicyLiveExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "name", policyName),
					resource.TestCheckResourceAttr(fullResourceLabel, "policy_type", "VersionPolicy"),
					resource.TestCheckResourceAttr(fullResourceLabel, "resource_name", resourceCrn),
					resource.TestCheckResourceAttr(fullResourceLabel, "mode", "AUDIT"),
					resource.TestCheckResourceAttr(fullResourceLabel, "action", "DENY"),
					resource.TestCheckResourceAttr(fullResourceLabel, "rules.#", "1"),
					resource.TestCheckResourceAttrSet(fullResourceLabel, "id"),
				),
			},
			{
				ResourceName:      fullResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckClientRequestPolicyLiveConfig(endpoint, resourceLabel, policyName, resourceCrn, apiKey, apiSecret, "ACTIVE", "request.client.version >= '3.6.0'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClientRequestPolicyLiveExists(fullResourceLabel),
					resource.TestCheckResourceAttr(fullResourceLabel, "mode", "ACTIVE"),
					resource.TestCheckResourceAttr(fullResourceLabel, "rules.0.match", "request.client.version >= '3.6.0'"),
				),
			},
			{
				ResourceName:      fullResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckClientRequestPolicyLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_client_request_policy" {
			continue
		}
		deletedPolicyName := rs.Primary.ID
		_, response, err := c.configurationControlV1Client.PoliciesConfigurationcontrolV1Api.GetConfigurationcontrolV1Policy(c.configurationControlV1ApiContext(context.Background()), deletedPolicyName).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil {
			return fmt.Errorf("client request policy (%q) still exists", rs.Primary.ID)
		}
		return err
	}
	return nil
}

func testAccCheckClientRequestPolicyLiveConfig(endpoint, resourceLabel, policyName, resourceCrn, apiKey, apiSecret, mode, match string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_client_request_policy" "%s" {
		name          = "%s"
		policy_type   = "VersionPolicy"
		resource_name = "%s"
		mode          = "%s"
		action        = "DENY"
		rules {
			name  = "require-recent-java-client"
			match = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, policyName, resourceCrn, mode, match)
}

func testAccCheckClientRequestPolicyLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s client request policy has not been found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s client request policy", n)
		}
		return nil
	}
}
