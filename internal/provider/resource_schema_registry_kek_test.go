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
	kekResourceScenarioName        = "confluent_schema_registry_kek Resource Lifecycle"
	scenarioStateKekHasBeenCreated = "A new kek has been just created"
	scenarioStateKekHasBeenUpdated = "A new kek has been just updated"
	createKekUrlPath               = "/dek-registry/v1/keks"
	kekLabel                       = "confluent_schema_registry_kek.mykek"
)

func TestAccKek(t *testing.T) {
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

	createKekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_kek/kek.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createKekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKekHasBeenCreated).
		WillReturn(
			string(createKekResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenCreated).
		WillReturn(
			string(createKekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateKekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_kek/updated_kek.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenCreated).
		WillSetStateTo(scenarioStateKekHasBeenUpdated).
		WillReturn(
			string(updateKekResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenUpdated).
		WillReturn(
			string(updateKekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
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
				Config: kekResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kekLabel, "id", "111/testkek"),
					resource.TestCheckResourceAttr(kekLabel, "name", "testkek"),
					resource.TestCheckResourceAttr(kekLabel, "kms_type", "aws-kms"),
					resource.TestCheckResourceAttr(kekLabel, "kms_key_id", "kmsKeyId"),
					resource.TestCheckResourceAttr(kekLabel, "shared", "false"),
					resource.TestCheckResourceAttr(kekLabel, "hard_delete", "false"),
					resource.TestCheckResourceAttr(kekLabel, "doc", ""),
				),
			},
			{
				Config: kekResourceUpdatedConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kekLabel, "id", "111/testkek"),
					resource.TestCheckResourceAttr(kekLabel, "name", "testkek"),
					resource.TestCheckResourceAttr(kekLabel, "kms_type", "aws-kms"),
					resource.TestCheckResourceAttr(kekLabel, "kms_key_id", "kmsKeyId"),
					resource.TestCheckResourceAttr(kekLabel, "shared", "false"),
					resource.TestCheckResourceAttr(kekLabel, "hard_delete", "false"),
					resource.TestCheckResourceAttr(kekLabel, "doc", "new description"),
				),
			},
		},
	})
}

func kekResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "111"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_schema_registry_kek" "mykek" {
	  name = "testkek"
	  kms_type = "aws-kms"
	  kms_key_id = "kmsKeyId"
	  shared = false
	}

 	`, mockServerUrl)
}

func kekResourceUpdatedConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "111"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_schema_registry_kek" "mykek" {
	  name = "testkek"
	  kms_type = "aws-kms"
	  kms_key_id = "kmsKeyId"
	  doc = "new description"
	  shared = false
	}
 	`, mockServerUrl)
}
