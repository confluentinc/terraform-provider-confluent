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
	kekDataSourceScenarioName = "confluent_schema_registry_kek Data Source Lifecycle"
	kekUrlPath                = "/dek-registry/v1/keks/testkek"
	testKekName               = "testkek"
	kekDataSourceLabel        = "data.confluent_schema_registry_kek.kek"
)

func TestAccDataSourceSchemaRegistryKek(t *testing.T) {
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

	readKekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_kek/kek.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readKekResponse),
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
				Config: testAccCheckDataSourceKekDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kekDataSourceLabel, "id", "111/testkek"),
					resource.TestCheckResourceAttr(kekDataSourceLabel, "name", "testkek"),
					resource.TestCheckResourceAttr(kekDataSourceLabel, "kms_type", "aws-kms"),
					resource.TestCheckResourceAttr(kekDataSourceLabel, "kms_key_id", "kmsKeyId"),
					resource.TestCheckResourceAttr(kekDataSourceLabel, "shared", "false"),
					resource.TestCheckResourceAttr(kekDataSourceLabel, "hard_delete", "false"),
				),
			},
		},
	})
}

func testAccCheckDataSourceKekDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  schema_registry_id = "111"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "11"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret    = "1/1/1/4N/1"    # optionally use SCHEMA_REGISTRY_API_SECRET env var
	}
	data "confluent_schema_registry_kek" "kek" {
		name = "%s"
	}
	`, mockServerUrl, testKekName)
}
