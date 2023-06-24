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
	"context"
	"fmt"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	tfImporterResourceScenarioName = "confluent_tf_importer Resource Lifecycle"
	tfImporterResourceLabel        = "test_importer_resource_label"
)

func TestAccResourceTfImporter(t *testing.T) {
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

	readServiceAccounts, _ := ioutil.ReadFile("../testdata/service_account/read_sas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		InScenario(tfImporterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readServiceAccounts),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_created_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(tfImporterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullTfImporterResourceLabel := fmt.Sprintf("confluent_tf_importer.%s", tfImporterResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceTfImporterConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullTfImporterResourceLabel),
					//resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramId, saId),
				),
			},
		},
	})
}

func testAccCheckResourceTfImporterConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_tf_importer" "%s" {
        resources = ["confluent_service_account"]
	}
	`, mockServerUrl, tfImporterResourceLabel)
}
