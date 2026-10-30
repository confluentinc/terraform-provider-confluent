// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const switchoverEndpointsDataSourceLabel = "data.confluent_switchover_endpoints.main"

// The stub requires the switchover_pair query parameter, so the test also proves the optional
// pair filter is forwarded to the API.
func TestAccDataSourceSwitchoverEndpoints(t *testing.T) {
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

	listResponse, _ := os.ReadFile("../testdata/switchover/list_endpoints.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverEndpointsUrlPath)).
		WithQueryParam("environment", wiremock.EqualTo("env-abc123")).
		WithQueryParam("switchover_pair", wiremock.EqualTo("sw-abc123")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listSwitchoverPageSize))).
		WillReturn(
			string(listResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSwitchoverEndpointsConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_pair_id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.#", "1"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.id", "se-abc123"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.display_name", "prod-kafka-dr-endpoint"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.parent_resource_crn", switchoverEndpointParentResourceCrn),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.target", "east-platt"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.phase", "READY"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.#", "2"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.1.name", "east-platt"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.1.hostname", "east-platt.dr.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.1.connection_type", "PRIVATELINK"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.1.endpoint_filter.0.type", "private"),
					resource.TestCheckResourceAttr(switchoverEndpointsDataSourceLabel, "switchover_endpoints.0.endpoints.1.endpoint_filter.0.network_crn", switchoverEndpointEastNetworkCrn),
				),
			},
		},
	})
}

func testAccCheckDataSourceSwitchoverEndpointsConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	data "confluent_switchover_endpoints" "main" {
		environment_crn    = "%s"
		switchover_pair_id = "sw-abc123"
	}
	`, mockServerUrl, switchoverPairEnvironmentCrn)
}
