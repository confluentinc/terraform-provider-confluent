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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSubjectCompatibilityLevelSchemaWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedSubjectCompatibilityLevelResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_created_subject_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		InScenario(subjectCompatibilityLevelDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSubjectCompatibilityLevelResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectCompatibilityLevelDataSourceConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSubjectCompatibilityLevelDataSourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "compatibility_level", testSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "%", strconv.Itoa(testNumberOfSubjectCompatibilityLevelDataSourceAttributes)),
				),
			},
		},
	})
}

func testAccCheckSubjectCompatibilityLevelDataSourceConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      schema_registry_rest_endpoint = "%s"
      schema_registry_api_key = "%s"
      schema_registry_api_secret = "%s"
      schema_registry_id = "%s"
    }
	data "confluent_subject_config" "%s" {
	  subject_name = "%s"
	}
	`, confluentCloudBaseUrl, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testStreamGovernanceClusterId, testSchemaResourceLabel, testSubjectName)
}
