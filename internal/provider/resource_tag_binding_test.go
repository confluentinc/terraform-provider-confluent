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
	tagBindingResourceScenarioName        = "confluent_tag_binding Data Source Lifecycle"
	scenarioStateTagBindingHasBeenCreated = "A new tag binding has been just created"
	createTagBindingUrlPath               = "/catalog/v1/entity/tags"
	readCreatedTagBindingUrlPath          = "/catalog/v1/entity/type/sr_schema/name/lsrc-8wrx70:.:100001/tags"
	deleteCreatedTagBindingUrlPath        = "/catalog/v1/entity/type/sr_schema/name/lsrc-8wrx70:.:100001/tags/tag1"
	tagBindingLabel                       = "confluent_tag_binding.main"
)

func TestAccTagBinding(t *testing.T) {
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

	createTagBindingResponse, _ := ioutil.ReadFile("../testdata/tag/create_tag_binding.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createTagBindingUrlPath)).
		InScenario(tagBindingResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateTagBindingHasBeenCreated).
		WillReturn(
			string(createTagBindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readTagBindingResponse, _ := ioutil.ReadFile("../testdata/tag/read_tag_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedTagBindingUrlPath)).
		InScenario(tagBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateTagBindingHasBeenCreated).
		WillReturn(
			string(readTagBindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(deleteCreatedTagBindingUrlPath)).
		InScenario(tagBindingResourceScenarioName).
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
				Config: tagBindingResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagBindingLabel, "tag_name", "tag1"),
					resource.TestCheckResourceAttr(tagBindingLabel, "entity_name", "lsrc-8wrx70:.:100001"),
					resource.TestCheckResourceAttr(tagBindingLabel, "entity_type", "sr_schema"),
					resource.TestCheckResourceAttr(tagBindingLabel, "id", "xxx/tag1/lsrc-8wrx70:.:100001/sr_schema"),
				),
			},
		},
	})
}

func tagBindingResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_tag_binding" "main" {
      tag_name = "tag1"
	  entity_name = "lsrc-8wrx70:.:100001"
	  entity_type = "sr_schema"
	}

 	`, mockServerUrl)
}
