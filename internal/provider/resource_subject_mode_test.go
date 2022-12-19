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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSubjectMode(t *testing.T) {
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

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSubjectModeTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSubjectModeDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectModeConfig(confluentCloudBaseUrl, mockSubjectModeTestServerUrl, testSubjectMode, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectModeExists(fullSubjectModeResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "mode", testSubjectMode),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "rest_endpoint", mockSubjectModeTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "%", testNumberOfSubjectModeResourceAttributes),
				),
			},
			{
				Config: testAccCheckSubjectModeConfig(confluentCloudBaseUrl, mockSubjectModeTestServerUrl, testUpdatedSubjectMode, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectModeExists(fullSubjectModeResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "mode", testUpdatedSubjectMode),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "rest_endpoint", mockSubjectModeTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.key", testSchemaRegistryUpdatedKey),
					resource.TestCheckResourceAttr(fullSubjectModeResourceLabel, "credentials.0.secret", testSchemaRegistryUpdatedSecret),
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

func testAccCheckSubjectModeConfig(confluentCloudBaseUrl, mockServerUrl, mode, schemaRegistryKey, schemaRegistrySecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_subject_mode" "%s" {
      credentials {	
        key = "%s"	
        secret = "%s"	
	  }
      rest_endpoint = "%s"
	  schema_registry_cluster {
        id = "%s"
      }
	
	  subject_name = "%s"
	  mode = "%s"
	}
	`, confluentCloudBaseUrl, testSubjectModeResourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSubjectName, mode)
}
