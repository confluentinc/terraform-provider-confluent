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
				Config: testAccCheckSwitchoverPairFailoverConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "switchover_pair_id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "active_member", "east"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "failover_type", "CLEAN"),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverPairFailoverResourceLabel, "phase", "SWITCHING"),
				),
			},
		},
	})
}

func testAccCheckSwitchoverPairFailoverConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	resource "confluent_switchover_pair_failover" "main" {
		switchover_pair_id = "sw-abc123"
		active_member      = "east"
		failover_type      = "CLEAN"
		environment_crn    = "%s"
	}
	`, mockServerUrl, switchoverPairEnvironmentCrn)
}
