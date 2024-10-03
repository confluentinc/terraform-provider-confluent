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
	scenarioStateCertificatePoolHasBeenCreated = "The new certificate_pool has been just created"
	scenarioStateCertificatePoolHasBeenUpdated = "The new certificate_pool has been updated"
	CertificatePoolScenarioName                = "confluent_certificate_pool Resource Lifecycle"

	certificatePoolUrlPath       = "/iam/v2/certificate-authorities/op-abc123/identity-pools"
	certificatePoolId            = "pool-def456"
	certificatePoolResourceLabel = "confluent_certificate_pool.main"
)

func TestAccCertificatePool(t *testing.T) {
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
	createCertificatePoolResponse, _ := ioutil.ReadFile("../testdata/certificate_pool/create_certificate_pool.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(certificatePoolUrlPath)).
		InScenario(CertificatePoolScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCertificatePoolHasBeenCreated).
		WillReturn(
			string(createCertificatePoolResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId))).
		InScenario(CertificatePoolScenarioName).
		WhenScenarioStateIs(scenarioStateCertificatePoolHasBeenCreated).
		WillReturn(
			string(createCertificatePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCertificatePoolResponse, _ := ioutil.ReadFile("../testdata/certificate_pool/read_updated_certificate_pool.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId))).
		InScenario(CertificatePoolScenarioName).
		WhenScenarioStateIs(scenarioStateCertificatePoolHasBeenCreated).
		WillSetStateTo(scenarioStateCertificatePoolHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificatePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId))).
		InScenario(CertificatePoolScenarioName).
		WhenScenarioStateIs(scenarioStateCertificatePoolHasBeenUpdated).
		WillReturn(
			string(readUpdatedCertificatePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", certificatePoolUrlPath, certificatePoolId))).
		InScenario(CertificatePoolScenarioName).
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
				Config: testAccCheckResourceCertificatePoolConfig(mockServerUrl, "example-description", "C=='Canada' && O=='Confluent'"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "id", certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "display_name", "my-certificate-pool"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "description", "example-description"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "external_identifier", "UID"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "filter", "C=='Canada' && O=='Confluent'"),
				),
			},
			{
				Config: testAccCheckResourceCertificatePoolConfig(mockServerUrl, "example-description-new", "S=='Spain' && O=='Confluent'"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "id", certificatePoolId),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "display_name", "my-certificate-pool"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "description", "example-description-new"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "external_identifier", "UID"),
					resource.TestCheckResourceAttr(certificatePoolResourceLabel, "filter", "S=='Spain' && O=='Confluent'"),
				),
			},
		},
	})
}

func testAccCheckResourceCertificatePoolConfig(mockServerUrl, description, filter string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_certificate_pool" "main" {
		certificate_authority {
		    id = "op-abc123"
		}
		display_name = "my-certificate-pool"
		description = "%s"
		external_identifier = "UID"
		filter = "%s"
	}
	`, mockServerUrl, description, filter)
}
