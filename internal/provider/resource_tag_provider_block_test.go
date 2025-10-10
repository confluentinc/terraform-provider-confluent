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
	"testing"
)

const (
	tagResourceScenarioName        = "confluent_tag Resource Lifecycle"
	scenarioStateTagHasBeenCreated = "A new tag has been just created"
	scenarioStateTagHasBeenPending = "A new tag has been just pending"
	scenarioStateTagHasBeenUpdated = "A new tag has been just updated"
	scenarioStateTagHasBeenDeleted = "A new tag has been just deleted"
	createTagUrlPath               = "/catalog/v1/types/tagdefs"
	readCreatedTagUrlPath          = "/catalog/v1/types/tagdefs/test1"
	tagLabel                       = "confluent_tag.mytag"
)

func TestAccTagWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: tagResourceConfigWithEnhancedProviderBlock(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagLabel, "id", fmt.Sprintf("%s/test1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(tagLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagLabel, "description", "test1Description"),
					resource.TestCheckResourceAttr(tagLabel, "version", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.0", "cf_entity"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(tagLabel, "rest_endpoint"),
				),
			},
			{
				Config: tagResourceConfigWithEnhancedSchemaRegistryProviderBlock(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagLabel, "id", fmt.Sprintf("%s/test1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(tagLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagLabel, "description", "test1Description"),
					resource.TestCheckResourceAttr(tagLabel, "version", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.0", "cf_entity"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(tagLabel, "rest_endpoint"),
				),
			},
			{
				Config: tagResourceUpdatedConfigWithEnhancedProviderBlock(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagLabel, "id", fmt.Sprintf("%s/test1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(tagLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(tagLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagLabel, "description", "test1UpdatedDescription"),
					resource.TestCheckResourceAttr(tagLabel, "version", "2"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagLabel, "entity_types.0", "cf_entity"),
					resource.TestCheckResourceAttr(tagLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(tagLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(tagLabel, "rest_endpoint"),
				),
			},
		},
	})
}

func tagResourceConfigWithEnhancedProviderBlock(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "%s"
	  catalog_rest_endpoint = "%s"               # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "%s"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "%s"
 	}
 	resource "confluent_tag" "mytag" {
	  name = "test1"
	  description = "test1Description"
	}

 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}

func tagResourceConfigWithEnhancedSchemaRegistryProviderBlock(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "%s"
	  schema_registry_rest_endpoint = "%s"       # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "%s"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "%s"
 	}
 	resource "confluent_tag" "mytag" {
	  name = "test1"
	  description = "test1Description"
	}

 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}

func tagResourceUpdatedConfigWithEnhancedProviderBlock(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "%s"
	  catalog_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "%s"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "%s"
 	}
 	resource "confluent_tag" "mytag" {
	  name = "test1"
	  description = "test1UpdatedDescription"
	}
 	`, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}
