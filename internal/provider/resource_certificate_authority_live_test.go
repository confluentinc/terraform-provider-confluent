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

func TestAccCertificateAuthorityLive(t *testing.T) {
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

	// Note: Certificate Authority requires an actual valid certificate chain for testing
	// This test uses a sample self-signed certificate for demonstration purposes
	// In production, use a valid certificate from your CA
	certChain := os.Getenv("TEST_CERTIFICATE_CHAIN")
	if certChain == "" {
		t.Skip("TEST_CERTIFICATE_CHAIN environment variable must be set with a valid PEM certificate chain for Certificate Authority live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	caDisplayName := fmt.Sprintf("tf-live-ca-%d", randomSuffix)
	caResourceLabel := "test_live_certificate_authority"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCertificateAuthorityLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCertificateAuthorityLiveConfig(endpoint, caResourceLabel, caDisplayName, certChain, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertificateAuthorityLiveExists(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "display_name", caDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "description", "Test Certificate Authority for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "certificate_chain_filename", "ca-cert.pem"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "fingerprints.#"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "expiration_dates.#"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "serial_numbers.#"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"certificate_chain", // Sensitive field not returned by API
				},
			},
			{
				Config: testAccCheckCertificateAuthorityUpdateLiveConfig(endpoint, caResourceLabel, caDisplayName, certChain, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCertificateAuthorityLiveExists(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "display_name", caDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel), "description", "Updated Certificate Authority description"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_certificate_authority.%s", caResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"certificate_chain", // Sensitive field not returned by API
				},
			},
		},
	})
}

func testAccCheckCertificateAuthorityLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Certificate Authority is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_certificate_authority" {
			continue
		}
		deletedCAId := rs.Primary.ID
		req := c.caClient.CertificateAuthoritiesIamV2Api.GetIamV2CertificateAuthority(c.caApiContext(context.Background()), deletedCAId)
		deletedCA, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// If the error is equivalent to http.StatusNotFound, the Certificate Authority is destroyed.
			return nil
		} else if err == nil && deletedCA.Id != nil {
			// Otherwise return the error
			if *deletedCA.Id == rs.Primary.ID {
				return fmt.Errorf("Certificate Authority (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckCertificateAuthorityLiveConfig(endpoint, resourceLabel, displayName, certChain, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_authority" "%s" {
		display_name = "%s"
		description = "Test Certificate Authority for live testing"
		certificate_chain = <<EOT
%s
EOT
		certificate_chain_filename = "ca-cert.pem"
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, displayName, certChain)
}

func testAccCheckCertificateAuthorityUpdateLiveConfig(endpoint, resourceLabel, displayName, certChain, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_certificate_authority" "%s" {
		display_name = "%s"
		description = "Updated Certificate Authority description"
		certificate_chain = <<EOT
%s
EOT
		certificate_chain_filename = "ca-cert.pem"
	}
	`, endpoint, apiKey, apiSecret, resourceLabel, displayName, certChain)
}

func testAccCheckCertificateAuthorityLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Certificate Authority has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Certificate Authority", n)
		}

		return nil
	}
}

