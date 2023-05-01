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
	businessMetadataResourceScenarioName        = "confluent_business_metadata Data Source Lifecycle"
	scenarioStateBusinessMetadataHasBeenCreated = "A new business metadata has been just created"
	scenarioStateBusinessMetadataHasBeenUpdated = "A new business metadata has been just updated"
	createBusinessMetadataUrlPath               = "/catalog/v1/types/businessmetadatadefs"
	readCreatedBusinessMetadataUrlPath          = "/catalog/v1/types/businessmetadatadefs/bm"
	businessMetadataLabel                       = "confluent_business_metadata.main"
)

func TestAccBusinessMetadata(t *testing.T) {
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

	createBusinessMetadataResponse, _ := ioutil.ReadFile("../testdata/business_metadata/create_business_metadata.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createBusinessMetadataUrlPath)).
		InScenario(businessMetadataResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateBusinessMetadataHasBeenCreated).
		WillReturn(
			string(createBusinessMetadataResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	updateBusinessMetadataResponse, _ := ioutil.ReadFile("../testdata/business_metadata/update_business_metadata.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createBusinessMetadataUrlPath)).
		InScenario(businessMetadataResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataHasBeenCreated).
		WillSetStateTo(scenarioStateBusinessMetadataHasBeenUpdated).
		WillReturn(
			string(updateBusinessMetadataResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readBusinessMetadataResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_created_business_metadata.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataUrlPath)).
		InScenario(businessMetadataResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataHasBeenCreated).
		WillReturn(
			string(readBusinessMetadataResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedBusinessMetadataResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_updated_business_metadata.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataUrlPath)).
		InScenario(businessMetadataResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataHasBeenUpdated).
		WillReturn(
			string(readUpdatedBusinessMetadataResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readCreatedBusinessMetadataUrlPath)).
		InScenario(businessMetadataResourceScenarioName).
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
				Config: businessMetadataResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataLabel, paramId, "xxx/bm"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramDescription, "bm description"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramVersion, "1"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.#", paramAttributeDef), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramName), "attr1"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramName), "attr2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
				),
			},
			{
				Config: businessMetadataResourceUpdatedConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataLabel, paramId, "xxx/bm"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramDescription, "bm description"),
					resource.TestCheckResourceAttr(businessMetadataLabel, paramVersion, "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.#", paramAttributeDef), "3"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramName), "attr1"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.0.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramName), "attr2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.1.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s", paramAttributeDef, paramName), "attr3"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s", paramAttributeDef, paramIsOptional), "true"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataLabel, fmt.Sprintf("%s.2.%s.maxStrLength", paramAttributeDef, paramOptions), "5000")),
			},
		},
	})
}

func businessMetadataResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata" "main" {
	  name = "bm"
	  description = "bm description"
	  attribute_definition {
		name = "attr1"
	  }
	  attribute_definition {
		name = "attr2"
	  }
	}
 	`, mockServerUrl)
}

func businessMetadataResourceUpdatedConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata" "main" {
	  name = "bm"
	  description = "bm description"
	  attribute_definition {
		name = "attr1"
	  }
	  attribute_definition {
		name = "attr2"
	  }
	  attribute_definition {
        name = "attr3"
        is_optional = true
	  }
	}
 	`, mockServerUrl)
}
