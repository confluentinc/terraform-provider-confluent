//go:build live_test && (all || core)

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
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCertificatePoolLive(t *testing.T) {
	// Disable parallel execution to avoid conflicts with Certificate Authority test
	// Both tests may use the same certificate chain
	// t.Parallel()

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

	// Certificate Pool requires a Certificate Authority
	certificateAuthorityId := os.Getenv("TEST_CERTIFICATE_AUTHORITY_ID")
	certChain := os.Getenv("TEST_CERTIFICATE_CHAIN")
	if certificateAuthorityId == "" && certChain == "" {
		t.Skip("Either TEST_CERTIFICATE_AUTHORITY_ID or TEST_CERTIFICATE_CHAIN must be set for Certificate Pool live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	caDisplayName := fmt.Sprintf("tf-live-ca-for-pool-%d", randomSuffix)
	poolDisplayName := fmt.Sprintf("tf-live-cert-pool-%d", randomSuffix)
	poolUpdatedDisplayName := fmt.Sprintf("tf-live-cert-pool-updated-%d", randomSuffix)
	caResourceLabel := "test_live_certificate_authority_for_pool"
	poolResourceLabel := "test_live_certificate_pool"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCertificatePoolLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCertificatePoolLiveConfig(endpoint, caResourceLabel, poolResourceLabel, caDisplayName, poolDisplayName, certChain, certificateAuthorityId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertificatePoolLiveExists(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "display_name", poolDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "description", "Test Certificate Pool for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "external_identifier", "CN"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "filter", "O=='Confluent Test'"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "certificate_authority.0.id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)].Primary.ID
					caId := resources[fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)].Primary.Attributes["certificate_authority.0.id"]
					return fmt.Sprintf("%s/%s", caId, poolId), nil
				},
			},
			{
				Config: testAccCheckCertificatePoolUpdateLiveConfig(endpoint, caResourceLabel, poolResourceLabel, caDisplayName, poolUpdatedDisplayName, certChain, certificateAuthorityId, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertificatePoolLiveExists(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "display_name", poolUpdatedDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "description", "Updated Certificate Pool description"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "external_identifier", "CN"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel), "filter", "CN=='test-ca.confluent.io'"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)].Primary.ID
					caId := resources[fmt.Sprintf("confluent_certificate_pool.%s", poolResourceLabel)].Primary.Attributes["certificate_authority.0.id"]
					return fmt.Sprintf("%s/%s", caId, poolId), nil
				},
			},
		},
	})
}

func testAccCheckCertificatePoolLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Certificate Pool is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_certificate_pool" {
			continue
		}
		deletedPoolId := rs.Primary.ID
		certificateAuthorityId := rs.Primary.Attributes["certificate_authority.0.id"]
		req := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPool(c.caApiContext(context.Background()), certificateAuthorityId, deletedPoolId)
		deletedPool, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// If the error is equivalent to http.StatusNotFound, the Certificate Pool is destroyed.
			return nil
		} else if err == nil && deletedPool.Id != nil {
			// Otherwise return the error
			if *deletedPool.Id == rs.Primary.ID {
				return fmt.Errorf("Certificate Pool (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckCertificatePoolLiveConfig(endpoint, caResourceLabel, poolResourceLabel, caDisplayName, poolDisplayName, certChain, certificateAuthorityId, apiKey, apiSecret string) string {
	if certificateAuthorityId != "" {
		// Use existing Certificate Authority
		return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_pool" "%s" {
		certificate_authority {
			id = "%s"
		}
		display_name = "%s"
		description = "Test Certificate Pool for live testing"
		external_identifier = "CN"
		filter = "O=='Confluent Test'"
	}
	`, endpoint, apiKey, apiSecret, poolResourceLabel, certificateAuthorityId, poolDisplayName)
	}

	// Create Certificate Authority first
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_authority" "%s" {
		display_name = "%s"
		description = "Test CA for Certificate Pool"
		certificate_chain = <<EOT
%s
EOT
		certificate_chain_filename = "ca-cert.pem"
	}

	resource "confluent_certificate_pool" "%s" {
		certificate_authority {
			id = confluent_certificate_authority.%s.id
		}
		display_name = "%s"
		description = "Test Certificate Pool for live testing"
		external_identifier = "CN"
		filter = "O=='Confluent Test'"
	}
	`, endpoint, apiKey, apiSecret, caResourceLabel, caDisplayName, certChain, poolResourceLabel, caResourceLabel, poolDisplayName)
}

func testAccCheckCertificatePoolUpdateLiveConfig(endpoint, caResourceLabel, poolResourceLabel, caDisplayName, poolDisplayName, certChain, certificateAuthorityId, apiKey, apiSecret string) string {
	if certificateAuthorityId != "" {
		// Use existing Certificate Authority
		return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_pool" "%s" {
		certificate_authority {
			id = "%s"
		}
		display_name = "%s"
		description = "Updated Certificate Pool description"
		external_identifier = "CN"
		filter = "CN=='test-ca.confluent.io'"
	}
	`, endpoint, apiKey, apiSecret, poolResourceLabel, certificateAuthorityId, poolDisplayName)
	}

	// Create Certificate Authority first
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_authority" "%s" {
		display_name = "%s"
		description = "Test CA for Certificate Pool"
		certificate_chain = <<EOT
%s
EOT
		certificate_chain_filename = "ca-cert.pem"
	}

	resource "confluent_certificate_pool" "%s" {
		certificate_authority {
			id = confluent_certificate_authority.%s.id
		}
		display_name = "%s"
		description = "Updated Certificate Pool description"
		external_identifier = "CN"
		filter = "CN=='test-ca.confluent.io'"
	}
	`, endpoint, apiKey, apiSecret, caResourceLabel, caDisplayName, certChain, poolResourceLabel, caResourceLabel, poolDisplayName)
}

func testAccCheckCertificatePoolLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Certificate Pool has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Certificate Pool", n)
		}

		return nil
	}
}

