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
	scenarioStateSubjectModeHasBeenCreated = "A new subject mode has been just created"
	scenarioStateSubjectModeHasBeenUpdated = "The subject mode has been updated"
	scenarioStateSubjectModeHasBeenDeleted = "The subject mode has been deleted"
	subjectModeScenarioName                = "confluent_subject_mode Resource Lifecycle"

	testSubjectModeResourceLabel = "test_subject_mode_resource_label"
	testSubjectMode              = "READWRITE"
	testUpdatedSubjectMode       = "READONLY"

	testNumberOfSubjectModeResourceAttributes = "6"
)

// TODO: APIF-1990
var mockSubjectModeTestServerUrl = ""

var fullSubjectModeResourceLabel = fmt.Sprintf("confluent_subject_mode.%s", testSubjectModeResourceLabel)
var updateSubjectModePath = fmt.Sprintf("/mode/%s", testSubjectName)

func TestAccSubjectModeWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSubjectModeTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSubjectModeTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createSubjectModeResponse, _ := ioutil.ReadFile("../testdata/subject_mode/read_created_subject_mode.json")
	createSubjectModeStub := wiremock.Put(wiremock.URLPathEqualTo(updateSubjectModePath)).
		InScenario(subjectModeScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSubjectModeHasBeenCreated).
		WillReturn(
			string(createSubjectModeResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSubjectModeStub)

	readCreatedSubjectModesResponse, _ := ioutil.ReadFile("../testdata/subject_mode/read_created_subject_mode.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectModePath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(subjectModeScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectModeHasBeenCreated).
		WillReturn(
			string(readCreatedSubjectModesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(updateSubjectModePath)).
		InScenario(subjectModeScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectModeHasBeenCreated).
		WillSetStateTo(scenarioStateSubjectModeHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSubjectModesResponse, _ := ioutil.ReadFile("../testdata/subject_mode/read_updated_subject_mode.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSubjectModePath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(subjectModeScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectModeHasBeenUpdated).
		WillReturn(
			string(readUpdatedSubjectModesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSubjectModeStub := wiremock.Delete(wiremock.URLPathEqualTo(updateSubjectModePath)).
		InScenario(subjectModeScenarioName).
		WhenScenarioStateIs(scenarioStateSubjectModeHasBeenUpdated).
		WillSetStateTo(scenarioStateSubjectModeHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteSubjectModeStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSubjectModeDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSubjectModeTestServerUrl, testSubjectMode),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectModeExists(fullSubjectModeResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "mode", testSubjectMode),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "%", testNumberOfSubjectModeResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSubjectModeResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckSubjectModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSubjectModeTestServerUrl, testUpdatedSubjectMode),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectModeExists(fullSubjectModeResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "mode", testUpdatedSubjectMode),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSubjectModeResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "%", testNumberOfSubjectModeResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSubjectModeResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSubjectModeStub, fmt.Sprintf("PUT (CREATE) %s", updateSubjectModePath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteSubjectModeStub, fmt.Sprintf("DELETE %s", updateSubjectModePath), expectedCountOne)
}

func testAccCheckSubjectModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl, mode string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  schema_registry_api_key = "%s"
	  schema_registry_api_secret = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_id = "%s"
	}
	resource "confluent_subject_mode" "%s" {
	  subject_name = "%s"
	  mode = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryKey, testSchemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSubjectModeResourceLabel, testSubjectName, mode)
}

func testAccCheckSubjectModeExists(n string) resource.TestCheckFunc {
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

func testAccCheckSubjectModeDestroy(s *terraform.State) error {
	return nil
}
