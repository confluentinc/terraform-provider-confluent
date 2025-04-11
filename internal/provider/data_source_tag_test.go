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
	tagDataSourceScenarioName = "confluent_tag Data Source Lifecycle"
	tagUrlPath                = "/catalog/v1/types/tagdefs/ttt6"
	testTagName               = "ttt6"
	tagDataSourceLabel        = "data.confluent_tag.tag"
)

func TestAccDataSourceTag(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	readTagResponse, _ := ioutil.ReadFile("../testdata/tag/read_tag.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(tagUrlPath)).
		InScenario(tagDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readTagResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceTagDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagDataSourceLabel, "id", "111/test1"),
					resource.TestCheckResourceAttr(tagDataSourceLabel, "name", "test1"),
					resource.TestCheckResourceAttr(tagDataSourceLabel, "description", "test1Description"),
					resource.TestCheckResourceAttr(tagDataSourceLabel, "version", "1"),
					resource.TestCheckResourceAttr(tagDataSourceLabel, "entity_types.#", "1"),
					resource.TestCheckResourceAttr(tagDataSourceLabel, "entity_types.0", "cf_entity"),
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

func testAccCheckDataSourceTagDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  schema_registry_id = "111"
	  catalog_rest_endpoint = "%s" # optionally use CATALOG_REST_ENDPOINT env var
	  schema_registry_api_key       = "11"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret    = "1/1/1/4N/1"    # optionally use SCHEMA_REGISTRY_API_SECRET env var
	}
	data "confluent_tag" "tag" {
		name = "%s"
	}
	`, mockServerUrl, testTagName)
}
