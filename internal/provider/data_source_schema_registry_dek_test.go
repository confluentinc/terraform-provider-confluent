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
	dekDataSourceScenarioName = "confluent_schema_registry_dek Data Source Lifecycle"
	dekDataSourceLabel        = "data.confluent_schema_registry_dek.dek"
)

func TestAccDataSourceSchemaRegistryDek(t *testing.T) {
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

	readDekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_dek/dek.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dekUrlPath)).
		InScenario(dekDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readDekResponse),
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
				Config: testAccCheckDataSourceDekDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dekDataSourceLabel, "id", "111/testkek/ts/1/AES256_GCM"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "kek_name", "testkek"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "algorithm", "AES256_GCM"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "encrypted_key_material", "tm"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "subject_name", "ts"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "hard_delete", "false"),
					resource.TestCheckResourceAttr(dekDataSourceLabel, "key_material", ""),
				),
			},
		},
	})
}

func testAccCheckDataSourceDekDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  schema_registry_id = "111"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "11"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret    = "1/1/1/4N/1"    # optionally use SCHEMA_REGISTRY_API_SECRET env var
	}
	data "confluent_schema_registry_dek" "dek" {
		kek_name = "testkek"
	    subject_name = "ts"
	}
	`, mockServerUrl)
}
