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

func TestAccSchemaRegistryClusterCompatibilityLevel(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaRegistryClusterCompatibilityLevelTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaRegistryClusterCompatibilityLevelTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createSchemaRegistryClusterCompatibilityLevelResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_compatibility_level/read_created_schema_registry_cluster_compatibility_level.json")
	createSchemaRegistryClusterCompatibilityLevelStub := wiremock.Put(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(schemaRegistryClusterCompatibilityLevelScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenCreated).
		WillReturn(
			string(createSchemaRegistryClusterCompatibilityLevelResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSchemaRegistryClusterCompatibilityLevelStub)

	readCreatedSchemaRegistryClusterCompatibilityLevelsResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_compatibility_level/read_created_schema_registry_cluster_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(schemaRegistryClusterCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenCreated).
		WillReturn(
			string(readCreatedSchemaRegistryClusterCompatibilityLevelsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(schemaRegistryClusterCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenCreated).
		WillSetStateTo(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSchemaRegistryClusterCompatibilityLevelsResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_compatibility_level/read_updated_schema_registry_cluster_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(schemaRegistryClusterCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenUpdated).
		WillReturn(
			string(readUpdatedSchemaRegistryClusterCompatibilityLevelsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSchemaRegistryClusterCompatibilityLevelStub := wiremock.Delete(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(schemaRegistryClusterCompatibilityLevelScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenUpdated).
		WillSetStateTo(scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteSchemaRegistryClusterCompatibilityLevelStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSchemaRegistryClusterCompatibilityLevelTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterCompatibilityLevelDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterCompatibilityLevelConfig(confluentCloudBaseUrl, mockSchemaRegistryClusterCompatibilityLevelTestServerUrl, testSchemaRegistryClusterCompatibilityLevel, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterCompatibilityLevelExists(fullSchemaRegistryClusterCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "compatibility_level", testSchemaRegistryClusterCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "rest_endpoint", mockSchemaRegistryClusterCompatibilityLevelTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "%", testNumberOfSchemaRegistryClusterCompatibilityLevelResourceAttributes),
				),
			},
			{
				Config: testAccCheckSchemaRegistryClusterCompatibilityLevelConfig(confluentCloudBaseUrl, mockSchemaRegistryClusterCompatibilityLevelTestServerUrl, testUpdatedSchemaRegistryClusterCompatibilityLevel, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterCompatibilityLevelExists(fullSchemaRegistryClusterCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "compatibility_level", testUpdatedSchemaRegistryClusterCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "rest_endpoint", mockSchemaRegistryClusterCompatibilityLevelTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.key", testSchemaRegistryUpdatedKey),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.secret", testSchemaRegistryUpdatedSecret),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "%", testNumberOfSchemaRegistryClusterCompatibilityLevelResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaRegistryClusterCompatibilityLevelResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSchemaRegistryClusterCompatibilityLevelStub, fmt.Sprintf("PUT (CREATE) %s", updateSchemaRegistryClusterCompatibilityLevelPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteSchemaRegistryClusterCompatibilityLevelStub, fmt.Sprintf("DELETE %s", updateSchemaRegistryClusterCompatibilityLevelPath), expectedCountZero)
}

func testAccCheckSchemaRegistryClusterCompatibilityLevelConfig(confluentCloudBaseUrl, mockServerUrl, compatibilityLevel, schemaRegistryKey, schemaRegistrySecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_schema_registry_cluster_config" "%s" {
      credentials {	
        key = "%s"	
        secret = "%s"	
	  }
      rest_endpoint = "%s"
	  schema_registry_cluster {
        id = "%s"
      }
	
	  compatibility_level = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryClusterCompatibilityLevelResourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, compatibilityLevel)
}
