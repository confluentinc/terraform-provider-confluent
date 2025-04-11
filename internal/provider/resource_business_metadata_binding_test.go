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
	"time"
)

const (
	businessMetadataBindingResourceScenarioName        = "confluent_business_metadata_binding Resource Lifecycle"
	scenarioStateBusinessMetadataBindingHasBeenCreated = "A new business metadata binding has been just created"
	scenarioStateBusinessMetadataBindingHasBeenPending = "A new business metadata binding has been just pending"
	scenarioStateBusinessMetadataBindingHasBeenUpdated = "A new business metadata binding has been just updated"
	createBusinessMetadataBindingUrlPath               = "/catalog/v1/entity/businessmetadata"
	readCreatedBusinessMetadataBindingUrlPath          = "/catalog/v1/entity/type/kafka_topic/name/lsrc-8wrx70:lkc-m80307:topic_0/businessmetadata"
	deleteCreatedBusinessMetadataBindingUrlPath        = "/catalog/v1/entity/type/kafka_topic/name/lsrc-8wrx70:lkc-m80307:topic_0/businessmetadata/bm"
	businessMetadataBindingLabel                       = "confluent_business_metadata_binding.main"
)

func TestAccBusinessMetadataBinding(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	createBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/create_business_metadata_binding.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenPending).
		WillReturn(
			string(createBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenPending).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_created_business_metadata_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillReturn(
			string(readBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/update_business_metadata_binding.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenUpdated).
		WillReturn(
			string(updateBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readUpdatedBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_updated_business_metadata_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenUpdated).
		WillReturn(
			string(readUpdatedBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(deleteCreatedBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
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
				Config: businessMetadataBindingResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityName, "lsrc-8wrx70:lkc-m80307:topic_0"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramId, "xxx/bm/lsrc-8wrx70:lkc-m80307:topic_0/kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.%%", paramAttributes), "2"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
				),
			},
			{
				Config: businessMetadataBindingSchemaRegistryResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityName, "lsrc-8wrx70:lkc-m80307:topic_0"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramId, "xxx/bm/lsrc-8wrx70:lkc-m80307:topic_0/kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.%%", paramAttributes), "2"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
				),
			},
			{
				Config: updateBusinessMetadataBindingResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityName, "lsrc-8wrx70:lkc-m80307:topic_0"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramId, "xxx/bm/lsrc-8wrx70:lkc-m80307:topic_0/kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.%%", paramAttributes), "3"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr3", paramAttributes), "value3"),
				),
			},
		},
	})
}

func businessMetadataBindingResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  catalog_rest_endpoint = "%s" 	  # optionally use CATALOG_REST_ENDPOINT env var
	  schema_registry_api_key = "x"   # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata_binding" "main" {
	  business_metadata_name = "bm"
	  entity_name = "lsrc-8wrx70:lkc-m80307:topic_0"
	  entity_type = "kafka_topic"
	  attributes = {
		"attr1" = "value1"
		"attr2" = "value2"
	  }
	}
 	`, mockServerUrl)
}

func businessMetadataBindingSchemaRegistryResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" 	  # optionally use CATALOG_REST_ENDPOINT env var
	  schema_registry_api_key = "x"   # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata_binding" "main" {
	  business_metadata_name = "bm"
	  entity_name = "lsrc-8wrx70:lkc-m80307:topic_0"
	  entity_type = "kafka_topic"
	  attributes = {
		"attr1" = "value1"
		"attr2" = "value2"
	  }
	}
 	`, mockServerUrl)
}

func updateBusinessMetadataBindingResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
      catalog_rest_endpoint = "%s" 	  # optionally use CATALOG_REST_ENDPOINT env var
	  schema_registry_api_key = "x"   # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata_binding" "main" {
	  business_metadata_name = "bm"
	  entity_name = "lsrc-8wrx70:lkc-m80307:topic_0"
	  entity_type = "kafka_topic"
	  attributes = {
		"attr1" = "value1"
		"attr2" = "value2"
        "attr3" = "value3"
	  }
	}
 	`, mockServerUrl)
}
