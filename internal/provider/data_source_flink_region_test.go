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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceFlinkRegionScenarioName = "confluent_flink_region Data Source Lifecycle"
	flinkRegionResourceLabel          = "example"
	flinkRegionCloudProvider          = "AWS"
	flinkRegionCloudProviderRegion    = "us-east-1"
	flinkRegionKind                   = "Region"
	flinkRegionRestEndpoint           = "https://flink.us-east-1.aws.confluent.cloud"
	flinkRegionRestEndpointPrivate    = "https://flink.us-east-1.aws.private.confluent.cloud"
)

var fullFlinkRegionDataSourceLabel = fmt.Sprintf("data.confluent_flink_region.%s", flinkRegionResourceLabel)

func TestAccDataSourceFlinkRegion(t *testing.T) {
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

	readRegionsResponse, _ := ioutil.ReadFile("../testdata/flink_region/read_regions.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/regions")).
		InScenario(dataSourceFlinkRegionScenarioName).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listFlinkRegionsPageSize))).
		WithQueryParam("cloud", wiremock.EqualTo(flinkRegionCloudProvider)).
		WithQueryParam("region_name", wiremock.EqualTo(flinkRegionCloudProviderRegion)).
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
				Config: testAccCheckDataSourceFlinkRegionConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkRegionExists(fullFlinkRegionDataSourceLabel),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramId,
						strings.ToLower(fmt.Sprintf("%s.%s", flinkRegionCloudProvider, flinkRegionCloudProviderRegion))),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramCloud,
						flinkRegionCloudProvider),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramRegion,
						flinkRegionCloudProviderRegion),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramRestEndpoint, flinkRegionRestEndpoint),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramRestEndpointPrivate, flinkRegionRestEndpointPrivate),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramApiVersion, flinkComputePoolApiVersion),
					resource.TestCheckResourceAttr(fullFlinkRegionDataSourceLabel, paramKind, flinkRegionKind),
				),
			},
		},
	})
}

func testAccCheckDataSourceFlinkRegionConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_flink_region" "example" {
		cloud = "%s"
	  	region = "%s"
	}
	`, mockServerUrl, flinkRegionCloudProvider, flinkRegionCloudProviderRegion)
}

func testAccCheckFlinkRegionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s flink region has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s flink region", n)
		}

		return nil
	}
}
