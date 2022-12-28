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

func TestAccSchemaRegistryClusterMode(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaRegistryClusterModeTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaRegistryClusterModeTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createSchemaRegistryClusterModeResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_mode/read_created_schema_registry_cluster_mode.json")
	createSchemaRegistryClusterModeStub := wiremock.Put(wiremock.URLPathEqualTo(updateSchemaRegistryClusterModePath)).
		InScenario(schemaRegistryClusterModeScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaRegistryClusterModeHasBeenCreated).
		WillReturn(
			string(createSchemaRegistryClusterModeResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createSchemaRegistryClusterModeStub)

	readCreatedSchemaRegistryClusterModesResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_mode/read_created_schema_registry_cluster_mode.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSchemaRegistryClusterModePath)).
		InScenario(schemaRegistryClusterModeScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterModeHasBeenCreated).
		WillReturn(
			string(readCreatedSchemaRegistryClusterModesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(updateSchemaRegistryClusterModePath)).
		InScenario(schemaRegistryClusterModeScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterModeHasBeenCreated).
		WillSetStateTo(scenarioStateSchemaRegistryClusterModeHasBeenUpdated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedSchemaRegistryClusterModesResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_mode/read_updated_schema_registry_cluster_mode.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSchemaRegistryClusterModePath)).
		InScenario(schemaRegistryClusterModeScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterModeHasBeenUpdated).
		WillReturn(
			string(readUpdatedSchemaRegistryClusterModesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteSchemaRegistryClusterModeStub := wiremock.Delete(wiremock.URLPathEqualTo(updateSchemaRegistryClusterModePath)).
		InScenario(schemaRegistryClusterModeScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaRegistryClusterModeHasBeenUpdated).
		WillSetStateTo(scenarioStateSchemaRegistryClusterModeHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteSchemaRegistryClusterModeStub)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT", mockSchemaRegistryClusterModeTestServerUrl)
	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterModeDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterModeConfig(confluentCloudBaseUrl, mockSchemaRegistryClusterModeTestServerUrl, testSchemaRegistryClusterMode, testSchemaRegistryKey, testSchemaRegistrySecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeExists(fullSchemaRegistryClusterModeResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "mode", testSchemaRegistryClusterMode),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "rest_endpoint", mockSchemaRegistryClusterModeTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "%", testNumberOfSchemaRegistryClusterModeResourceAttributes),
				),
			},
			{
				Config: testAccCheckSchemaRegistryClusterModeConfig(confluentCloudBaseUrl, mockSchemaRegistryClusterModeTestServerUrl, testUpdatedSchemaRegistryClusterMode, testSchemaRegistryUpdatedKey, testSchemaRegistryUpdatedSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeExists(fullSchemaRegistryClusterModeResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "mode", testUpdatedSchemaRegistryClusterMode),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "rest_endpoint", mockSchemaRegistryClusterModeTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.key", testSchemaRegistryUpdatedKey),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.secret", testSchemaRegistryUpdatedSecret),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "%", testNumberOfSchemaRegistryClusterModeResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaRegistryClusterModeResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSchemaRegistryClusterModeStub, fmt.Sprintf("PUT (CREATE) %s", updateSchemaRegistryClusterModePath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteSchemaRegistryClusterModeStub, fmt.Sprintf("DELETE %s", updateSchemaRegistryClusterModePath), expectedCountZero)
}

func testAccCheckSchemaRegistryClusterModeConfig(confluentCloudBaseUrl, mockServerUrl, mode, schemaRegistryKey, schemaRegistrySecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_schema_registry_cluster_mode" "%s" {
      credentials {	
        key = "%s"	
        secret = "%s"	
	  }
      rest_endpoint = "%s"
	  schema_registry_cluster {
        id = "%s"
      }
	
	  mode = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryClusterModeResourceLabel, schemaRegistryKey, schemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, mode)
}
