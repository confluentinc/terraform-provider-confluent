// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

func TestAccCertificateAuthority(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createCertificateAuthorityResponse, _ := ioutil.ReadFile("../testdata/certificate_authority/create_certificate_authority.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(certificateAuthorityUrlPath)).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCertificateAuthorityResponse, _ := ioutil.ReadFile("../testdata/certificate_authority/read_updated_certificate_authority.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceCertificateAuthorityConfig(mockServerUrl, "example-description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "description", "example-description"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "certificate_chain_filename", "certificate.pem"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "fingerprints.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "fingerprints.*", "B1BC968BD4f49D622AA89A81F2150152A41D829C"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "expiration_dates.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "expiration_dates.*", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "serial_numbers.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8F6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "false"),
				),
			},
			{
				Config: testAccCheckResourceCertificateAuthorityConfig(mockServerUrl, "example-description-new"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "description", "example-description-new"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "certificate_chain_filename", "certificate.pem"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "fingerprints.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "fingerprints.*", "B1BC968BD4f49D622AA89A81F2150152A41D829C"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "expiration_dates.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "expiration_dates.*", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "serial_numbers.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8F6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "false"),
				),
			},
		},
	})
}

func TestAccCertificateAuthorityCrl(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createCertificateAuthorityResponse, _ := ioutil.ReadFile("../testdata/certificate_authority/create_certificate_authority_crl.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(certificateAuthorityUrlPath)).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCertificateAuthorityResponse, _ := ioutil.ReadFile("../testdata/certificate_authority/read_updated_certificate_authority_crl.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceCertificateAuthorityCrlConfig(mockServerUrl, "example.url"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "description", "example-description"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "certificate_chain_filename", "certificate.pem"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "fingerprints.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "fingerprints.*", "B1BC968BD4f49D622AA89A81F2150152A41D829C"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "expiration_dates.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "expiration_dates.*", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "serial_numbers.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8F6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "example.url"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "URL"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_updated_at", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "true"),
				),
			},
			{
				Config: testAccCheckResourceCertificateAuthorityCrlConfig(mockServerUrl, "example-2.url"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "description", "example-description"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "certificate_chain_filename", "certificate.pem"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "fingerprints.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "fingerprints.*", "B1BC968BD4f49D622AA89A81F2150152A41D829C"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "expiration_dates.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "expiration_dates.*", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "serial_numbers.#", "1"),
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8F6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "example-2.url"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "URL"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_updated_at", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "true"),
				),
			},
		},
	})
}

func TestAccCertificateAuthorityCrlChain(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createCertificateAuthorityResponse, _ := ioutil.ReadFile("../testdata/certificate_authority/create_certificate_authority_crl_local.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(certificateAuthorityUrlPath)).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(
			string(createCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceCertificateAuthorityCrlChainConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "description", "example-description"),
					// Backend stamped the synthetic placeholder — state must capture it,
					// and the DiffSuppressFunc on crl_url must absorb the resulting
					// state-vs-empty-config mismatch without producing drift.
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "Local file uploaded"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "LOCAL"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "true"),
				),
				// No ExpectNonEmptyPlan: true here — the test will fail if the
				// post-apply plan is non-empty, which is exactly what would happen
				// if the schema regressed back to Optional-only.
			},
		},
	})
}

// TestAccCertificateAuthorityRequireFlipToTrue verifies that flipping
// require_crl_on_client_certificate from false to true produces state
// matching the backend-populated CRL fields after apply (not the prior empty
// values), which requires SetNewComputed in CustomizeDiff.
func TestAccCertificateAuthorityRequireFlipToTrue(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()
	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	// Step 1 — Create response: require=false, no CRL populated.
	createResp, _ := ioutil.ReadFile("../testdata/certificate_authority/create_certificate_authority.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(certificateAuthorityUrlPath)).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(string(createResp), contentTypeJSONHeader, http.StatusCreated))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillReturn(string(createResp), contentTypeJSONHeader, http.StatusOK))

	// Step 2 — Update response: require=true, CRL fields populated by backend.
	updateResp, _ := ioutil.ReadFile("../testdata/certificate_authority/create_certificate_authority_crl_local.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenCreated).
		WillSetStateTo(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(string(updateResp), contentTypeJSONHeader, http.StatusOK))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WhenScenarioStateIs(scenarioStateCertificateAuthorityHasBeenUpdated).
		WillReturn(string(updateResp), contentTypeJSONHeader, http.StatusOK))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificateAuthorityUrlPath, certificateAuthorityId))).
		InScenario(CertificateAuthorityScenarioName).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			// Step 1 — Create with require=false. State should have empty CRL fields.
			{
				Config: testAccCheckResourceCertificateAuthorityRequireFlipConfig(mockServerUrl, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "false"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", ""),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", ""),
				),
			},
			// Step 2 — Update to require=true with crl_chain. State MUST end up
			// with the backend-populated CRL fields (NOT the empty values from
			// the prior state). This is the assertion that fails without the
			// SetNewComputed branch in CustomizeDiff.
			{
				Config: testAccCheckResourceCertificateAuthorityRequireFlipConfig(mockServerUrl, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "id", certificateAuthorityId),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "require_crl_on_client_certificate", "true"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "Local file uploaded"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "LOCAL"),
				),
			},
		},
	})
}

func testAccCheckResourceCertificateAuthorityRequireFlipConfig(mockServerUrl string, requireCrl bool) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_certificate_authority" "main" {
		display_name = "my-ca"
		description = "example-description"
		certificate_chain_filename = "certificate.pem"
		certificate_chain = "ABC123"
		crl_chain = "DEF456"
		require_crl_on_client_certificate = %t
	}
	`, mockServerUrl, requireCrl)
}

func testAccCheckResourceCertificateAuthorityCrlChainConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_certificate_authority" "main" {
		display_name = "my-ca"
		description = "example-description"
		certificate_chain_filename = "certificate.pem"
		certificate_chain = "ABC123"
		crl_chain = "DEF456"
		require_crl_on_client_certificate = true
	}
	`, mockServerUrl)
}

func testAccCheckResourceCertificateAuthorityConfig(mockServerUrl, description string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_certificate_authority" "main" {
		display_name = "my-ca"
		description = "%s"
		certificate_chain_filename = "certificate.pem"
		certificate_chain = "ABC123"
	}
	`, mockServerUrl, description)
}

func testAccCheckResourceCertificateAuthorityCrlConfig(mockServerUrl, crlUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_certificate_authority" "main" {
		display_name = "my-ca"
		description = "example-description"
		certificate_chain_filename = "certificate.pem"
		certificate_chain = "ABC123"
		crl_url = "%s"
		require_crl_on_client_certificate = true
	}
	`, mockServerUrl, crlUrl)
}
