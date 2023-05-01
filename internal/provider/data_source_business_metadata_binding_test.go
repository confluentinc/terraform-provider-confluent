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
	businessMetadataBindingDataSourceScenarioName = "confluent_business_metadata_binding Data Source Lifecycle"
	businessMetadataBindingDataSourceLabel        = "data.confluent_business_metadata_binding.main"
)

func TestAccDataSourceBusinessMetadataBinding(t *testing.T) {
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

	readBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_created_business_metadata_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingUrlPath)).
		InScenario(businessMetadataBindingDataSourceScenarioName).
		WillReturn(
			string(readBusinessMetadataBindingResponse),
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
				Config: testAccCheckDataSourceBusinessMetadataBindingDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, paramEntityName, "lsrc-8wrx70:lkc-m80307:topic_0"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, paramEntityType, "kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, paramId, "xxx/bm/lsrc-8wrx70:lkc-m80307:topic_0/kafka_topic"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, fmt.Sprintf("%s.%%", paramAttributes), "2"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingDataSourceLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
				),
			},
		},
	})
}

func testAccCheckDataSourceBusinessMetadataBindingDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	data "confluent_business_metadata_binding" "main" {
      business_metadata_name = "bm"
	  entity_name = "lsrc-8wrx70:lkc-m80307:topic_0"
	  entity_type = "kafka_topic"
	}

 	`, mockServerUrl)
}
