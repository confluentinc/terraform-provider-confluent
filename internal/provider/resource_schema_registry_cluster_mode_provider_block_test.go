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
	scenarioStateSchemaRegistryClusterModeHasBeenCreated = "A new subject mode has been just created"
	scenarioStateSchemaRegistryClusterModeHasBeenUpdated = "The subject mode has been updated"
	scenarioStateSchemaRegistryClusterModeHasBeenDeleted = "The subject mode has been deleted"
	schemaRegistryClusterModeScenarioName                = "confluent_schema_registry_cluster_mode Resource Lifecycle"

	testSchemaRegistryClusterModeResourceLabel = "test_subject_mode_resource_label"
	testSchemaRegistryClusterMode              = "READWRITE"
	testUpdatedSchemaRegistryClusterMode       = "READONLY"

	testNumberOfSchemaRegistryClusterModeResourceAttributes = "5"
)

// TODO: APIF-1990
var mockSchemaRegistryClusterModeTestServerUrl = ""

var fullSchemaRegistryClusterModeResourceLabel = fmt.Sprintf("confluent_schema_registry_cluster_mode.%s", testSchemaRegistryClusterModeResourceLabel)
var updateSchemaRegistryClusterModePath = fmt.Sprintf("/mode")

func TestAccSchemaRegistryClusterModeWithEnhancedProviderBlock(t *testing.T) {
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckSchemaRegistryClusterModeDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaRegistryClusterModeTestServerUrl, testSchemaRegistryClusterMode),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeExists(fullSchemaRegistryClusterModeResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "mode", testSchemaRegistryClusterMode),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "%", testNumberOfSchemaRegistryClusterModeResourceAttributes),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullSchemaRegistryClusterModeResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckSchemaRegistryClusterModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockSchemaRegistryClusterModeTestServerUrl, testUpdatedSchemaRegistryClusterMode),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterModeExists(fullSchemaRegistryClusterModeResourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "mode", testUpdatedSchemaRegistryClusterMode),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullSchemaRegistryClusterModeResourceLabel, "rest_endpoint"),
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

func testAccCheckSchemaRegistryClusterModeConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl, mode string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	  schema_registry_api_key = "%s"
	  schema_registry_api_secret = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_id = "%s"
	}
	resource "confluent_schema_registry_cluster_mode" "%s" {
	  mode = "%s"
	}
	`, confluentCloudBaseUrl, testSchemaRegistryKey, testSchemaRegistrySecret, mockServerUrl, testStreamGovernanceClusterId, testSchemaRegistryClusterModeResourceLabel, mode)
}

func testAccCheckSchemaRegistryClusterModeExists(n string) resource.TestCheckFunc {
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

func testAccCheckSchemaRegistryClusterModeDestroy(s *terraform.State) error {
	return nil
}
