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

const (
	switchoverPairsDataSourceLabel = "data.confluent_switchover_pairs.main"
	switchoverPairsLastPageToken   = "eyJpZCI6InN3LWFiYzEyMyJ9"
)

// The list spans two pages: the first page carries a next_page_token, the second does not. Both
// stubs match on page_size so the request shape is checked too; the page-two stub is registered
// last and matches on page_token, so WireMock prefers it for the follow-up request.
func TestAccDataSourceSwitchoverPairs(t *testing.T) {
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

	pageOne, _ := os.ReadFile("../testdata/switchover/list_pairs_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairsUrlPath)).
		WithQueryParam("environment", wiremock.EqualTo("env-abc123")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listSwitchoverPageSize))).
		WillReturn(
			string(pageOne),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	pageTwo, _ := os.ReadFile("../testdata/switchover/list_pairs_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairsUrlPath)).
		WithQueryParam("environment", wiremock.EqualTo("env-abc123")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listSwitchoverPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(switchoverPairsLastPageToken)).
		WillReturn(
			string(pageTwo),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSwitchoverPairsConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.#", "2"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.display_name", "prod-kafka-dr"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.active_member", "east"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.first_active", "west"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.failover_type", "PLANNED"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.phase", "READY_TO_FAILOVER"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.members.#", "2"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.members.0.name", "west"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.members.0.member_crn", switchoverPairWestMemberCrn),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.0.members.0.cloud", "AWS"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.1.id", "sw-def456"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.1.display_name", "staging-kafka-dr"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.1.active_member", "west"),
					resource.TestCheckResourceAttr(switchoverPairsDataSourceLabel, "switchover_pairs.1.phase", "PROVISIONING"),
				),
			},
		},
	})
}

func testAccCheckDataSourceSwitchoverPairsConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	data "confluent_switchover_pairs" "main" {
		environment_crn = "%s"
	}
	`, mockServerUrl, switchoverPairEnvironmentCrn)
}
