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
	dataSourceStreamGovernanceRegionScenarioName = "confluent_stream_governance_region Data Source Lifecycle"
	streamGovernanceRegionResourceLabel          = "example"
	streamGovernanceRegionBillingPackage         = "ESSENTIALS"
	streamGovernanceRegionCloudProvider          = "AWS"
	streamGovernanceRegionCloudProviderRegion    = "us-east-2"
)

var fullStreamGovernanceRegionDataSourceLabel = fmt.Sprintf("data.confluent_stream_governance_region.%s", streamGovernanceRegionResourceLabel)

func TestAccDataSourceStreamGovernanceRegion(t *testing.T) {
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

	readRegionsResponse, _ := ioutil.ReadFile("../testdata/stream_governance_region/read_regions.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/stream-governance/v2/regions")).
		InScenario(dataSourceStreamGovernanceRegionScenarioName).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listStreamGovernanceRegionsPageSize))).
		WithQueryParam("spec.cloud", wiremock.EqualTo(streamGovernanceRegionCloudProvider)).
		WithQueryParam("spec.packages", wiremock.EqualTo(streamGovernanceRegionBillingPackage)).
		WithQueryParam("spec.region_name", wiremock.EqualTo(streamGovernanceRegionCloudProviderRegion)).
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
				Config: testAccCheckDataSourceStreamGovernanceRegionConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStreamGovernanceRegionExists(fullStreamGovernanceRegionDataSourceLabel),
					resource.TestCheckResourceAttr(fullStreamGovernanceRegionDataSourceLabel, paramId, streamGovernanceClusterRegionId),
					resource.TestCheckResourceAttr(fullStreamGovernanceRegionDataSourceLabel, paramCloud, streamGovernanceRegionCloudProvider),
					resource.TestCheckResourceAttr(fullStreamGovernanceRegionDataSourceLabel, paramRegion, streamGovernanceRegionCloudProviderRegion),
					resource.TestCheckResourceAttr(fullStreamGovernanceRegionDataSourceLabel, paramPackage, streamGovernanceRegionBillingPackage),
				),
			},
		},
	})
}

func testAccCheckDataSourceStreamGovernanceRegionConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_stream_governance_region" "example" {
		cloud = "%s"
	  	region = "%s"
	  	package = "%s"
	}
	`, mockServerUrl, streamGovernanceRegionCloudProvider, streamGovernanceRegionCloudProviderRegion, streamGovernanceRegionBillingPackage)
}

func testAccCheckStreamGovernanceRegionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s stream governance region has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s stream governance region", n)
		}

		return nil
	}
}
