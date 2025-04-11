// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestAccTag(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	createTagResponse, _ := ioutil.ReadFile("../testdata/tag/create_tag.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTagHasBeenPending).
		WillReturn(
			string(createTagResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WhenScenarioStateIs(scenarioStateTagHasBeenPending).
		WillSetStateTo(scenarioStateTagHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	updateTagResponse, _ := ioutil.ReadFile("../testdata/tag/update_tag.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WhenScenarioStateIs(scenarioStateTagHasBeenCreated).
		WillSetStateTo(scenarioStateTagHasBeenUpdated).
		WillReturn(
			string(updateTagResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readTagResponse, _ := ioutil.ReadFile("../testdata/tag/read_tag.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WhenScenarioStateIs(scenarioStateTagHasBeenCreated).
		WillReturn(
			string(readTagResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedTagResponse, _ := ioutil.ReadFile("../testdata/tag/read_updated_tag.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WhenScenarioStateIs(scenarioStateTagHasBeenUpdated).
		WillReturn(
			string(readUpdatedTagResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readCreatedTagUrlPath)).
		InScenario(tagResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_KEY", testSchemaRegistryUpdatedKey)
	_ = os.Setenv("IMPORT_SCHEMA_REGISTRY_API_SECRET", testSchemaRegistryUpdatedSecret)
	_ = os.Setenv("IMPORT_CATALOG_REST_ENDPOINT", mockServerUrl)

	defer func() {
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_KEY")
		_ = os.Unsetenv("IMPORT_SCHEMA_REGISTRY_API_SECRET")
		_ = os.Unsetenv("IMPORT_CATALOG_REST_ENDPOINT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: tagResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagLabel, "id", fmt.Sprintf("%s/test1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(tagLabel, "rest_endpoint", mockServerUrl),
					resource.TestCheckResourceAttr(tagLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(tagLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagLabel, "description", "test1Description"),
					resource.TestCheckResourceAttr(tagLabel, "version", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.0", "cf_entity"),
				),
			},
			{
				Config: tagResourceUpdatedConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagLabel, "id", fmt.Sprintf("%s/test1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(tagLabel, "rest_endpoint", mockServerUrl),
					resource.TestCheckResourceAttr(tagLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(tagLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(tagLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagLabel, "description", "test1UpdatedDescription"),
					resource.TestCheckResourceAttr(tagLabel, "version", "2"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.0", "cf_entity"),
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

func tagResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	}
 	resource "confluent_tag" "mytag" {
      name        = "test1"
      description = "test1Description"

      schema_registry_cluster {
        id = "%s"
      }

      rest_endpoint = "%s"

      credentials {
        key    = "%s"
        secret = "%s"
      }
   }
 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}

func tagResourceUpdatedConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	}
 	resource "confluent_tag" "mytag" {
      name        = "test1"
      description = "test1UpdatedDescription"

      schema_registry_cluster {
        id = "%s"
      }

      rest_endpoint = "%s"

      credentials {
        key    = "%s"
        secret = "%s"
      }
   }
 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}
