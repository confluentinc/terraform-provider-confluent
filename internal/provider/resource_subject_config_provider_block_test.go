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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	scenarioStateSubjectCompatibilityLevelHasBeenCreated = "A new subject compatibility level has been just created"
	scenarioStateSubjectCompatibilityLevelHasBeenUpdated = "The subject compatibility level has been updated"
	scenarioStateSubjectCompatibilityLevelHasBeenDeleted = "The subject compatibility level has been deleted"
	subjectCompatibilityLevelScenarioName                = "confluent_subject_config Resource Lifecycle"

	testSubjectCompatibilityLevelResourceLabel = "test_subject_compatibility_level_resource_label"
	testSubjectCompatibilityLevel              = "FULL"
	testUpdatedSubjectCompatibilityLevel       = "BACKWARD_TRANSITIVE"

	testNumberOfSubjectCompatibilityLevelResourceAttributes = "6"
)

// TODO: APIF-1990
var mockSubjectCompatibilityLevelTestServerUrl = ""

var fullSubjectCompatibilityLevelResourceLabel = fmt.Sprintf("confluent_subject_config.%s", testSubjectCompatibilityLevelResourceLabel)
var updateSubjectCompatibilityLevelPath = fmt.Sprintf("/config/%s", testSubjectName)

func TestAccSubjectCompatibilityLevelWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSubjectCompatibilityLevelTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSubjectCompatibilityLevelTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createSubjectCompatibilityLevelResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_created_subject_compatibility_level.json")
	createSubjectCompatibilityLevelStub := wiremock.Put(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		InScenario(subjectCompatibilityLevelScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSubjectCompatibilityLevelHasBeenCreated).
		WillReturn(
			string(createSubjectCompatibilityLevelResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSubjectCompatibilityLevelStub)

	readCreatedSubjectCompatibilityLevelsResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_created_subject_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(subjectCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectCompatibilityLevelHasBeenCreated).
		WillReturn(
			string(readCreatedSubjectCompatibilityLevelsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		InScenario(subjectCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectCompatibilityLevelHasBeenCreated).
		WillSetStateTo(scenarioStateSubjectCompatibilityLevelHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSubjectCompatibilityLevelsResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_updated_subject_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(subjectCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectCompatibilityLevelHasBeenUpdated).
		WillReturn(
			string(readUpdatedSubjectCompatibilityLevelsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSubjectCompatibilityLevelStub := wiremock.Delete(wiremock.URLPathEqualTo(updateSubjectCompatibilityLevelPath)).
		InScenario(subjectCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectCompatibilityLevelHasBeenUpdated).
		WillSetStateTo(scenarioStateSubjectCompatibilityLevelHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteSubjectCompatibilityLevelStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSubjectCompatibilityLevelDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testSubjectCompatibilityLevel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "%", testNumberOfSubjectCompatibilityLevelResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSubjectCompatibilityLevelResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckSubjectCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testUpdatedSubjectCompatibilityLevel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testUpdatedSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "%", testNumberOfSubjectCompatibilityLevelResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSubjectCompatibilityLevelResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSubjectCompatibilityLevelStub, fmt.Sprintf("PUT (CREATE) %s", updateSubjectCompatibilityLevelPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteSubjectCompatibilityLevelStub, fmt.Sprintf("DELETE %s", updateSubjectCompatibilityLevelPath), expectedCountOne)
}

func testAccCheckSubjectCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl, compatibilityLevel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  schema_registry_api_key = "%s"
	  schema_registry_api_secret = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_id = "%s"
	}
	resource "confluent_subject_config" "%s" {
	  subject_name = "%s"
	  compatibility_level = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryKey, testSchemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSubjectCompatibilityLevelResourceLabel, testSubjectName, compatibilityLevel)
}

func testAccCheckSubjectCompatibilityLevelExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s schema has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s schema", n)
		}

		return nil
	}
}

func testAccCheckSubjectCompatibilityLevelDestroy(s *terraform.State) error {
	return nil
}
