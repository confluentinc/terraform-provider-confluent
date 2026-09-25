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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	switchoverPairFailoverResourceLabel = "confluent_switchover_pair_failover.main"
	switchoverPairFailoverUrlPath       = "/switchover/v1/switchover-pairs/sw-abc123:failover"
)

func TestAccSwitchoverPairFailover(t *testing.T) {
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

	failoverResponse, _ := os.ReadFile("../testdata/switchover/failover_pair.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(switchoverPairFailoverUrlPath)).
		WillReturn(
			string(failoverResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		WillReturn(
			string(failoverResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				// failover_type omitted: create sends the PLANNED default and records it in state.
				Config: testAccCheckSwitchoverPairFailoverConfig(mockServerUrl, `active_member      = "east"`, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "switchover_pair_id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "active_member", "east"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "failover_type", "PLANNED"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "phase", "UPDATING"),
				),
			},
			{
				// Explicit RESTORE with no member: a deliberate change, so the action re-triggers.
				Config: testAccCheckSwitchoverPairFailoverConfig(mockServerUrl, "", `failover_type      = "RESTORE"`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckNoResourceAttr(switchoverPairFailoverResourceLabel, "active_member"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "failover_type", "RESTORE"),
				),
			},
			{
				// Regression: after a RESTORE apply, a config that omits failover_type again must NOT plan a
				// replacement back to PLANNED (that would re-fire a failover on a routine apply). With a
				// schema Default of "PLANNED" this step planned a -/+; Optional+Computed keeps "RESTORE".
				Config:             testAccCheckSwitchoverPairFailoverConfig(mockServerUrl, "", ""),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

// activeMemberLine and failoverTypeLine are whole HCL lines, or "" to omit the attribute.
func testAccCheckSwitchoverPairFailoverConfig(mockServerUrl, activeMemberLine, failoverTypeLine string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	resource "confluent_switchover_pair_failover" "main" {
		switchover_pair_id = "sw-abc123"
		%s
		%s
		environment_crn    = "%s"
	}
	`, mockServerUrl, activeMemberLine, failoverTypeLine, switchoverPairEnvironmentCrn)
}
