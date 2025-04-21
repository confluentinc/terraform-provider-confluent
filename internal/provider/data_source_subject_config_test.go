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

const (
	subjectCompatibilityLevelDataSourceScenarioName           = "confluent_subject_config Data Source Lifecycle"
	testNumberOfSubjectCompatibilityLevelDataSourceAttributes = 7
)

var fullSubjectCompatibilityLevelDataSourceLabel = fmt.Sprintf("data.confluent_subject_config.%s", testSchemaResourceLabel)

func TestAccDataSubjectCompatibilityLevelSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockSchemaTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)

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
				Config: testAccCheckSubjectCompatibilityLevelDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSubjectCompatibilityLevelDataSourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "compatibility_level", testSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "compatibility_group", testSubjectCompatibilityGroup),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelDataSourceLabel, "%", strconv.Itoa(testNumberOfSubjectCompatibilityLevelDataSourceAttributes)),
				),
			},
		},
	})
	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})
}

func testAccCheckSubjectCompatibilityLevelDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_subject_config" "%s" {
	  schema_registry_cluster {
        id = "%s"
      }
      rest_endpoint = "%s"
      credentials {
        key = "%s"
        secret = "%s"
	  }
	  subject_name = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testSubjectName)
}
