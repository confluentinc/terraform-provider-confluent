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

func TestAccSubjectCompatibilityLevel(t *testing.T) {
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

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSubjectCompatibilityLevelTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSubjectCompatibilityLevelDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testSubjectCompatibilityLevel, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "rest_endpoint", mockSubjectCompatibilityLevelTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "%", testNumberOfSubjectCompatibilityLevelResourceAttributes),
				),
			},
			{
				Config: testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testUpdatedSubjectCompatibilityLevel, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testUpdatedSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "rest_endpoint", mockSubjectCompatibilityLevelTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.key", testSchemaRegistryUpdatedKey),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.secret", testSchemaRegistryUpdatedSecret),
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

func testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockServerUrl, compatibilityLevel, schemaRegistryKey, schemaRegistrySecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_subject_config" "%s" {
      credentials {	
        key = "%s"	
        secret = "%s"	
	  }
      rest_endpoint = "%s"
	  schema_registry_cluster {
        id = "%s"
      }
	
	  subject_name = "%s"
	  compatibility_level = "%s"
	}
	`, confluentCloudBaseUrl, testSubjectCompatibilityLevelResourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSubjectName, compatibilityLevel)
}
