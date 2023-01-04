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
	scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenCreated = "A new subject mode has been just created"
	scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenUpdated = "The subject mode has been updated"
	scenarioStateSchemaRegistryClusterCompatibilityLevelHasBeenDeleted = "The subject mode has been deleted"
	schemaRegistryClusterCompatibilityLevelScenarioName                = "confluent_schema_registry_cluster_config Resource Lifecycle"

	testSchemaRegistryClusterCompatibilityLevelResourceLabel = "test_subject_compatibility_level_resource_label"
	testSchemaRegistryClusterCompatibilityLevel              = "FULL"
	testUpdatedSchemaRegistryClusterCompatibilityLevel       = "BACKWARD_TRANSITIVE"

	testNumberOfSchemaRegistryClusterCompatibilityLevelResourceAttributes = "5"
)

// TODO: APIF-1990
var mockSchemaRegistryClusterCompatibilityLevelTestServerUrl = ""

var fullSchemaRegistryClusterCompatibilityLevelResourceLabel = fmt.Sprintf("confluent_schema_registry_cluster_config.%s", testSchemaRegistryClusterCompatibilityLevelResourceLabel)
var updateSchemaRegistryClusterCompatibilityLevelPath = fmt.Sprintf("/config")

func TestAccSchemaRegistryClusterCompatibilityLevelWithEnhancedProviderBlock(t *testing.T) {
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterCompatibilityLevelDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaRegistryClusterCompatibilityLevelTestServerUrl, testSchemaRegistryClusterCompatibilityLevel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterCompatibilityLevelExists(fullSchemaRegistryClusterCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "compatibility_level", testSchemaRegistryClusterCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "%", testNumberOfSchemaRegistryClusterCompatibilityLevelResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaRegistryClusterCompatibilityLevelResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckSchemaRegistryClusterCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaRegistryClusterCompatibilityLevelTestServerUrl, testUpdatedSchemaRegistryClusterCompatibilityLevel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterCompatibilityLevelExists(fullSchemaRegistryClusterCompatibilityLevelResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "compatibility_level", testUpdatedSchemaRegistryClusterCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterCompatibilityLevelResourceLabel, "rest_endpoint"),
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

func testAccCheckSchemaRegistryClusterCompatibilityLevelConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl, compatibilityLevel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  schema_registry_api_key = "%s"
	  schema_registry_api_secret = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_id = "%s"
	}
	resource "confluent_schema_registry_cluster_config" "%s" {
	  compatibility_level = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryKey, testSchemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSchemaRegistryClusterCompatibilityLevelResourceLabel, compatibilityLevel)
}

func testAccCheckSchemaRegistryClusterCompatibilityLevelExists(n string) resource.TestCheckFunc {
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

func testAccCheckSchemaRegistryClusterCompatibilityLevelDestroy(s *terraform.State) error {
	return nil
}
