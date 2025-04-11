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
	tagBindingDataSourceScenarioName = "confluent_tag_binding Data Source Lifecycle"
	tagBindingDataSourceLabel        = "data.confluent_tag_binding.main"
)

func TestAccDataSourceTagBinding(t *testing.T) {
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

	readTagBindingResponse, _ := ioutil.ReadFile("../testdata/tag/read_tag_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedTagBindingUrlPath)).
		InScenario(tagBindingDataSourceScenarioName).
		WillReturn(
			string(readTagBindingResponse),
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
				Config: testAccCheckDataSourceTagBindingDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tagBindingDataSourceLabel, "tag_name", "tag1"),
					resource.TestCheckResourceAttr(tagBindingDataSourceLabel, "entity_name", "lsrc-8wrx70:.:100001"),
					resource.TestCheckResourceAttr(tagBindingDataSourceLabel, "entity_type", "sr_schema"),
					resource.TestCheckResourceAttr(tagBindingDataSourceLabel, "id", "xxx/tag1/lsrc-8wrx70:.:100001/sr_schema"),
				),
			},
		},
	})
}

func testAccCheckDataSourceTagBindingDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  catalog_rest_endpoint = "%s" # optionally use CATALOG_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	data "confluent_tag_binding" "main" {
      tag_name = "tag1"
	  entity_name = "lsrc-8wrx70:.:100001"
	  entity_type = "sr_schema"
	}

 	`, mockServerUrl)
}
