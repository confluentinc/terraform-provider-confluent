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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	entityAttributesResourceScenarioName        = "confluent_catalog_entity_attributes Resource Lifecycle"
	scenarioStateEntityAttributesHasBeenCreated = "A new entity attributes has been just created"
	scenarioStateEntityAttributesHasBeenUpdated = "A new entity attributes has been just updated"
	createEntityAttributesUrlPath               = "/catalog/v1/entity"
	readCreatedEntityAttributesUrlPath          = "/catalog/v1/entity/type/kafka_topic/name/lkc-15xq83:topic_0"
	deleteCreatedEntityAttributesUrlPath        = "/catalog/v1/entity"
	entityAttributesLabel                       = "confluent_catalog_entity_attributes.main"
	testDataCatalogSchemaRegistryClusterID      = "lsrc-8wrx70"
	testAttributesToImport                      = "owner,description,ownerEmail"
)

func TestAccCatalogEntityAttributesWithEnhancedProviderBlock(t *testing.T) {
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

	createEntityAttributesResponse, _ := ioutil.ReadFile("../testdata/entity_attributes/create_response.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createEntityAttributesUrlPath)).
		InScenario(entityAttributesResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateEntityAttributesHasBeenCreated).
		WillReturn(
			string(createEntityAttributesResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readEntityAttributesResponse, _ := ioutil.ReadFile("../testdata/entity_attributes/create_entity_attributes.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedEntityAttributesUrlPath)).
		InScenario(entityAttributesResourceScenarioName).
		WhenScenarioStateIs(scenarioStateEntityAttributesHasBeenCreated).
		WillReturn(
			string(readEntityAttributesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateEntityAttributesResponse, _ := ioutil.ReadFile("../testdata/entity_attributes/create_response.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createEntityAttributesUrlPath)).
		InScenario(entityAttributesResourceScenarioName).
		WhenScenarioStateIs(scenarioStateEntityAttributesHasBeenCreated).
		WillSetStateTo(scenarioStateEntityAttributesHasBeenUpdated).
		WillReturn(
			string(updateEntityAttributesResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readUpdatedEntityAttributesResponse, _ := ioutil.ReadFile("../testdata/entity_attributes/update_entity_attributes.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedEntityAttributesUrlPath)).
		InScenario(entityAttributesResourceScenarioName).
		WhenScenarioStateIs(scenarioStateEntityAttributesHasBeenUpdated).
		WillReturn(
			string(readUpdatedEntityAttributesResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(deleteCreatedEntityAttributesUrlPath)).
		InScenario(entityAttributesResourceScenarioName).
		WhenScenarioStateIs(scenarioStateEntityAttributesHasBeenUpdated).
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
				Config: entityAttributesResourceConfigWithEnhancedProviderBlock(testDataCatalogSchemaRegistryClusterID, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityName, "lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramId, "kafka_topic/lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.%%", paramAttributes), "3"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.owner", paramAttributes), "dev"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.description", paramAttributes), "test_des"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.ownerEmail", paramAttributes), "dev@gmail.com"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      entityAttributesLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					entityTypeAndEntityName := resources[entityAttributesLabel].Primary.ID
					return testDataCatalogSchemaRegistryClusterID + "/" + entityTypeAndEntityName + "/" + testAttributesToImport, nil
				},
			},
			{
				Config: entityAttributesResourceConfigWithEnhancedProviderBlock(testDataCatalogSchemaRegistryClusterID, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityName, "lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramId, "kafka_topic/lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.%%", paramAttributes), "3"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.owner", paramAttributes), "dev"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.description", paramAttributes), "test_des"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.ownerEmail", paramAttributes), "dev@gmail.com"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      entityAttributesLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					entityTypeAndEntityName := resources[entityAttributesLabel].Primary.ID
					return testDataCatalogSchemaRegistryClusterID + "/" + entityTypeAndEntityName + "/" + testAttributesToImport, nil
				},
			},
			{
				Config: updateEntityAttributesResourceConfigWithEnhancedProviderBlock(testDataCatalogSchemaRegistryClusterID, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityName, "lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(entityAttributesLabel, paramId, "kafka_topic/lkc-15xq83:topic_0"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.%%", paramAttributes), "3"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.owner", paramAttributes), "dev"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.description", paramAttributes), "test_des"),
					resource.TestCheckResourceAttr(entityAttributesLabel, fmt.Sprintf("%s.ownerEmail", paramAttributes), "dev2@gmail.com"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      entityAttributesLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					entityTypeAndEntityName := resources[entityAttributesLabel].Primary.ID
					return testDataCatalogSchemaRegistryClusterID + "/" + entityTypeAndEntityName + "/" + testAttributesToImport, nil
				},
			},
		},
	})
}

func entityAttributesResourceConfigWithEnhancedProviderBlock(testDataCatalogSchemaRegistryClusterID, mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "%s"
	  catalog_rest_endpoint = "%s"          # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"   # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_catalog_entity_attributes" "main" {
	  entity_name = "lkc-15xq83:topic_0"
	  entity_type = "kafka_topic"
	  attributes = {
		"owner" : "dev",
		"description": "test_des",
		"ownerEmail": "dev@gmail.com"
	  }
	}
 	`, testDataCatalogSchemaRegistryClusterID, mockServerUrl)
}

func updateEntityAttributesResourceConfigWithEnhancedProviderBlock(testDataCatalogSchemaRegistryClusterID, mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "%s"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_catalog_entity_attributes" "main" {
	  entity_name = "lkc-15xq83:topic_0"
	  entity_type = "kafka_topic"
	  attributes = {
		"owner" : "dev",
		"description": "test_des",
		"ownerEmail": "dev2@gmail.com"
	  }
	}
 	`, testDataCatalogSchemaRegistryClusterID, mockServerUrl)
}
