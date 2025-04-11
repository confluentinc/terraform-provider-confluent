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
	subjectModeDataSourceScenarioName           = "confluent_subject_mode Data Source Lifecycle"
	testNumberOfSubjectModeDataSourceAttributes = 6
)

var fullSubjectModeDataSourceLabel = fmt.Sprintf("data.confluent_subject_mode.%s", testSchemaResourceLabel)

func TestAccDataSubjectModeSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockSchemaTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)

	readCreatedSubjectModeResponse, _ := ioutil.ReadFile("../testdata/subject_mode/read_created_subject_mode.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectModePath)).
		InScenario(subjectModeDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSubjectModeResponse),
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
				Config: testAccCheckSubjectModeDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSubjectModeDataSourceLabel),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "mode", testSubjectMode),
					resource.TestCheckResourceAttr(fullSubjectModeDataSourceLabel, "%", strconv.Itoa(testNumberOfSubjectModeDataSourceAttributes)),
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

func testAccCheckSubjectModeDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_subject_mode" "%s" {
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
