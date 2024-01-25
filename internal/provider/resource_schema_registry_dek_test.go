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
	dekResourceScenarioName        = "confluent_schema_registry_dek Resource Lifecycle"
	scenarioStateDekHasBeenCreated = "A new dek has been just created"
	createDekUrlPath               = "/dek-registry/v1/keks/testkek/deks"
	dekLabel                       = "confluent_schema_registry_dek.mydek"
	dekUrlPath                     = "/dek-registry/v1/keks/testkek/deks/ts/versions/1"
)

func TestAccDek(t *testing.T) {
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

	createDekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_dek/dek.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createDekUrlPath)).
		InScenario(dekResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateDekHasBeenCreated).
		WillReturn(
			string(createDekResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dekUrlPath)).
		InScenario(dekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateDekHasBeenCreated).
		WillReturn(
			string(createDekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(dekUrlPath)).
		InScenario(dekResourceScenarioName).
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
				Config: dekResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dekLabel, "id", "111/testkek/ts/1/AES256_GCM"),
					resource.TestCheckResourceAttr(dekLabel, "kek_name", "testkek"),
					resource.TestCheckResourceAttr(dekLabel, "algorithm", "AES256_GCM"),
					resource.TestCheckResourceAttr(dekLabel, "encrypted_key_material", "tm"),
					resource.TestCheckResourceAttr(dekLabel, "subject_name", "ts"),
					resource.TestCheckResourceAttr(dekLabel, "hard_delete", "true"),
					resource.TestCheckResourceAttr(dekLabel, "key_material", ""),
				),
			},
		},
	})
}

func dekResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "111"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_schema_registry_dek" "mydek" {
	  kek_name = "testkek"
	  subject_name = "ts"
	  encrypted_key_material = "tm"
	}

 	`, mockServerUrl)
}
