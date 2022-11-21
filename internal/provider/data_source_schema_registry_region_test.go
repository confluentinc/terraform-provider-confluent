// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceSchemaRegistryRegionScenarioName = "confluent_schema_registry_region Data Source Lifecycle"
	schemaRegistryRegionResourceLabel          = "example"
	schemaRegistryRegionBillingPackage         = "ESSENTIALS"
	schemaRegistryRegionCloudProvider          = "AWS"
	schemaRegistryRegionCloudProviderRegion    = "us-east-2"
)

var fullSchemaRegistryRegionDataSourceLabel = fmt.Sprintf("data.confluent_schema_registry_region.%s", schemaRegistryRegionResourceLabel)

func TestAccDataSourceSchemaRegistryRegion(t *testing.T) {
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

	readRegionsResponse, _ := ioutil.ReadFile("../testdata/schema_registry_region/read_regions.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v2/regions")).
		InScenario(dataSourceSchemaRegistryRegionScenarioName).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listSchemaRegistryRegionsPageSize))).
		WithQueryParam("spec.cloud", wiremock.EqualTo(schemaRegistryRegionCloudProvider)).
		WithQueryParam("spec.packages", wiremock.EqualTo(schemaRegistryRegionBillingPackage)).
		WithQueryParam("spec.region_name", wiremock.EqualTo(schemaRegistryRegionCloudProviderRegion)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readRegionsResponse),
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
				Config: testAccCheckDataSourceSchemaRegistryRegionConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryRegionExists(fullSchemaRegistryRegionDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryRegionDataSourceLabel, paramId, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryRegionDataSourceLabel, paramCloud, schemaRegistryRegionCloudProvider),
					resource.TestCheckResourceAttr(fullSchemaRegistryRegionDataSourceLabel, paramRegion, schemaRegistryRegionCloudProviderRegion),
					resource.TestCheckResourceAttr(fullSchemaRegistryRegionDataSourceLabel, paramPackage, schemaRegistryRegionBillingPackage),
				),
			},
		},
	})
}

func testAccCheckDataSourceSchemaRegistryRegionConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_schema_registry_region" "example" {
		cloud = "%s"
	  	region = "%s"
	  	package = "%s"
	}
	`, mockServerUrl, schemaRegistryRegionCloudProvider, schemaRegistryRegionCloudProviderRegion, schemaRegistryRegionBillingPackage)
}

func testAccCheckSchemaRegistryRegionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s schema registry region has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s schema registry region", n)
		}

		return nil
	}
}
