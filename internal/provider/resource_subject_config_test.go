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

	mockSubjectCompatibilityLevelTestServerUrl := wiremockContainer.URI
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
				Config: testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testSubjectCompatibilityLevel, testSubjectCompatibilityGroup, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_group", testSubjectCompatibilityGroup),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "normalize", "true"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "alias", ""),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "rest_endpoint", mockSubjectCompatibilityLevelTestServerUrl),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "%", testNumberOfSubjectCompatibilityLevelResourceAttributes),
				),
			},
			{
				Config: testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockSubjectCompatibilityLevelTestServerUrl, testUpdatedSubjectCompatibilityLevel, testSubjectCompatibilityGroup, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullSubjectCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testSubjectName)),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "subject_name", testSubjectName),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_level", testUpdatedSubjectCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "compatibility_group", testSubjectCompatibilityGroup),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "normalize", "true"),
					resource.TestCheckResourceAttr(fullSubjectCompatibilityLevelResourceLabel, "alias", ""),
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

func TestAccSubjectConfigWithAlias(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSubjectConfigTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSubjectConfigTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	testAliasSubjectName := "orders-alias-value"
	testAliasTarget := "orders-original-subject-value"
	testUpdatedAliasTarget := "orders-new-target-subject-value"
	aliasSubjectConfigPath := fmt.Sprintf("/config/%s", testAliasSubjectName)
	aliasScenarioName := "confluent_subject_config Alias Resource Lifecycle"
	aliasResourceLabel := "test_subject_config_alias"

	createSubjectConfigWithAliasResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_created_subject_config_with_alias.json")
	createSubjectConfigWithAliasStub := wiremock.Put(wiremock.URLPathEqualTo(aliasSubjectConfigPath)).
		InScenario(aliasScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo("AliasSubjectConfigCreated").
		WillReturn(
			string(createSubjectConfigWithAliasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSubjectConfigWithAliasStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(aliasSubjectConfigPath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(aliasScenarioName).
		WhenScenarioStateIs("AliasSubjectConfigCreated").
		WillReturn(
			string(createSubjectConfigWithAliasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(aliasSubjectConfigPath)).
		InScenario(aliasScenarioName).
		WhenScenarioStateIs("AliasSubjectConfigCreated").
		WillSetStateTo("AliasSubjectConfigUpdated").
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSubjectConfigWithAliasResponse, _ := ioutil.ReadFile("../testdata/subject_compatibility_level/read_updated_subject_config_with_alias.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(aliasSubjectConfigPath)).
		WithQueryParam("defaultToGlobal", wiremock.EqualTo("true")).
		InScenario(aliasScenarioName).
		WhenScenarioStateIs("AliasSubjectConfigUpdated").
		WillReturn(
			string(readUpdatedSubjectConfigWithAliasResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSubjectConfigWithAliasStub := wiremock.Delete(wiremock.URLPathEqualTo(aliasSubjectConfigPath)).
		InScenario(aliasScenarioName).
		WhenScenarioStateIs("AliasSubjectConfigUpdated").
		WillSetStateTo("AliasSubjectConfigDeleted").
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteSubjectConfigWithAliasStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistrySecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSubjectConfigTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

	fullAliasResourceLabel := fmt.Sprintf("confluent_subject_config.%s", aliasResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSubjectCompatibilityLevelDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSubjectConfigWithAliasConfig(confluentCloudBaseUrl, mockSubjectConfigTestServerUrl, aliasResourceLabel, testAliasSubjectName, testAliasTarget, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullAliasResourceLabel),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testAliasSubjectName)),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "subject_name", testAliasSubjectName),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "alias", testAliasTarget),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "compatibility_level", "BACKWARD"),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "normalize", "false"),
				),
			},
			{
				Config: testAccCheckSubjectConfigWithAliasConfig(confluentCloudBaseUrl, mockSubjectConfigTestServerUrl, aliasResourceLabel, testAliasSubjectName, testUpdatedAliasTarget, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSubjectCompatibilityLevelExists(fullAliasResourceLabel),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "id", fmt.Sprintf("%s/%s", testStreamGovernanceClusterId, testAliasSubjectName)),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "subject_name", testAliasSubjectName),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "alias", testUpdatedAliasTarget),
					resource.TestCheckResourceAttr(fullAliasResourceLabel, "compatibility_level", "BACKWARD"),
				),
			},
			{
				ResourceName:      fullAliasResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSubjectConfigWithAliasStub, fmt.Sprintf("PUT (CREATE) %s", aliasSubjectConfigPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteSubjectConfigWithAliasStub, fmt.Sprintf("DELETE %s", aliasSubjectConfigPath), expectedCountOne)
}

func testAccCheckSubjectConfigWithAliasConfig(confluentCloudBaseUrl, mockServerUrl, resourceLabel, subjectName, aliasTarget, schemaRegistryKey, schemaRegistrySecret string) string {
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
	
	  subject_name        = "%s"
	  compatibility_level = "BACKWARD"
	  alias               = "%s"
	}
	`, confluentCloudBaseUrl, resourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, subjectName, aliasTarget)
}

func testAccCheckSubjectCompatibilityLevelConfig(confluentCloudBaseUrl, mockServerUrl, compatibilityLevel, compatibilityGroup, schemaRegistryKey, schemaRegistrySecret string) string {
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
	  compatibility_group = "%s"
	  normalize = true
	}
	`, confluentCloudBaseUrl, testSubjectCompatibilityLevelResourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSubjectName, compatibilityLevel, compatibilityGroup)
}
