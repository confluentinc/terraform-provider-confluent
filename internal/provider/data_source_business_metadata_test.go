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
	businessMetadataDataSourceScenarioName = "confluent_business_metadata Data Source Lifecycle"
	testBusinessMetadataName               = "bm"
	businessMetadataDataSourceLabel        = "data.confluent_business_metadata.main"
)

func TestAccDataSourceBusinessMetadata(t *testing.T) {
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

	readBusinessMetadataResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_created_business_metadata.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataUrlPath)).
		InScenario(businessMetadataDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readBusinessMetadataResponse),
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
				Config: testAccCheckDataSourceBusinessMetadataDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, paramId, "xxx/bm"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, paramName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, paramDescription, "bm description"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, paramVersion, "1"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.#", paramAttributeDef), "2"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramName), "attr1"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.0.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramName), "attr2"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramIsOptional), "false"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s", paramAttributeDef, paramType), "string"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s.%%", paramAttributeDef, paramOptions), "2"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s.applicableEntityTypes", paramAttributeDef, paramOptions), "[\"cf_entity\"]"),
					resource.TestCheckResourceAttr(businessMetadataDataSourceLabel, fmt.Sprintf("%s.1.%s.maxStrLength", paramAttributeDef, paramOptions), "5000"),
				),
			},
		},
	})
}

func testAccCheckDataSourceBusinessMetadataDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 	  schema_registry_id = "xxx"
	  catalog_rest_endpoint = "%s"   # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key = "x"  # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
	data "confluent_business_metadata" "main" {
		name = "%s"
	}
	`, mockServerUrl, testBusinessMetadataName)
}
