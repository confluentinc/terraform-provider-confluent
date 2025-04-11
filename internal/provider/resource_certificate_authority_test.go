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

	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	scenarioStateCertificateAuthorityHasBeenCreated = "The new certificate_authority has been just created"
	scenarioStateCertificateAuthorityHasBeenUpdated = "The new certificate_authority has been updated"
	CertificateAuthorityScenarioName                = "confluent_certificate_authority Resource Lifecycle"

	certificateAuthorityUrlPath       = "/iam/v2/certificate-authorities"
	certificateAuthorityId            = "op-abc123"
	certificateAuthorityResourceLabel = "confluent_certificate_authority.main"
)

func TestAccCertificateAuthority(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8f6EC7177FA4EE8C3705797"),
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
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8f6EC7177FA4EE8C3705797"),
				),
			},
		},
	})
}

func TestAccCertificateAuthorityCrl(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8f6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "example.url"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "URL"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_updated_at", "2017-07-21 17:32:28 +0000 UTC"),
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
					resource.TestCheckTypeSetElemAttr(certificateAuthorityResourceLabel, "serial_numbers.*", "219C542DE8f6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_url", "example-2.url"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_source", "URL"),
					resource.TestCheckResourceAttr(certificateAuthorityResourceLabel, "crl_updated_at", "2017-07-21 17:32:28 +0000 UTC"),
				),
			},
		},
	})
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
	}
	`, mockServerUrl, crlUrl)
}
