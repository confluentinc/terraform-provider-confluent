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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	switchoverPairResourceLabel   = "confluent_switchover_pair.main"
	switchoverPairDataSourceLabel = "data.confluent_switchover_pair.main"
	switchoverPairsUrlPath        = "/switchover/v1/switchover-pairs"
	switchoverPairReadUrlPath     = "/switchover/v1/switchover-pairs/sw-abc123"
	switchoverPairScenarioName    = "confluent_switchover_pair Resource Lifecycle"
	switchoverPairFailoverHook    = "switchover-pair-failover"

	scenarioStateSwitchoverPairIsProvisioning = "The switchover pair is provisioning"
	scenarioStateSwitchoverPairHasBeenCreated = "The switchover pair has been created"
	scenarioStateSwitchoverPairHasBeenUpdated = "The switchover pair has been updated"
	scenarioStateSwitchoverPairHasFailedOver  = "The switchover pair has failed over"
	scenarioStateSwitchoverPairHasBeenDeleted = "The switchover pair has been deleted"

	switchoverPairEnvironmentCrn = "crn://confluent.cloud/organization=org-abc/environment=env-abc123"
	switchoverPairWestMemberCrn  = "crn://confluent.cloud/organization=org-abc/environment=env-abc123/cloud-cluster=lkc-west01"
	switchoverPairEastMemberCrn  = "crn://confluent.cloud/organization=org-abc/environment=env-def456/cloud-cluster=lkc-east01"
)

func TestAccSwitchoverPair(t *testing.T) {
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

	// Create returns 202 with the pair still PROVISIONING; the provider polls GET until
	// READY_TO_FAILOVER. The first poll still sees PROVISIONING and advances the scenario, the
	// second sees the provisioned pair.
	provisioningPairResponse, _ := os.ReadFile("../testdata/switchover/provisioning_pair.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(switchoverPairsUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSwitchoverPairIsProvisioning).
		WillReturn(
			string(provisioningPairResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairIsProvisioning).
		WillSetStateTo(scenarioStateSwitchoverPairHasBeenCreated).
		WillReturn(
			string(provisioningPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	createPairResponse, _ := os.ReadFile("../testdata/switchover/create_pair.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenCreated).
		WillReturn(
			string(createPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedPairResponse, _ := os.ReadFile("../testdata/switchover/updated_pair.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenCreated).
		WillSetStateTo(scenarioStateSwitchoverPairHasBeenUpdated).
		WillReturn(
			string(updatedPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenUpdated).
		WillReturn(
			string(updatedPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// A failover (triggered outside this resource, e.g. by confluent_switchover_pair_failover) moves
	// active_member from "west" to "east" while first_active stays "west". The test flips the
	// scenario into this state out of band, via this hook stub, before the plan-only step below.
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(wiremockScenarioHookPath+switchoverPairFailoverHook)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenUpdated).
		WillSetStateTo(scenarioStateSwitchoverPairHasFailedOver).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	failedOverPairResponse, _ := os.ReadFile("../testdata/switchover/failed_over_pair.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasFailedOver).
		WillReturn(
			string(failedOverPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasFailedOver).
		WillSetStateTo(scenarioStateSwitchoverPairHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeletedPairResponse, _ := os.ReadFile("../testdata/switchover/read_deleted_pair.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenDeleted).
		WillReturn(
			string(readDeletedPairResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSwitchoverPairConfig(mockServerUrl, "prod-kafka-dr"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "display_name", "prod-kafka-dr"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "active_member", "west"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "first_active", "west"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "failover_type", "PLANNED"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "phase", "READY_TO_FAILOVER"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.#", "2"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.0.name", "west"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.0.member_crn", switchoverPairWestMemberCrn),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.0.cloud", "AWS"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.1.name", "east"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "members.1.member_crn", switchoverPairEastMemberCrn),
				),
			},
			{
				Config: testAccCheckSwitchoverPairConfig(mockServerUrl, "prod-kafka-dr-v2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "display_name", "prod-kafka-dr-v2"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "environment_crn", switchoverPairEnvironmentCrn),
				),
			},
			{
				// Regression test: after a failover flips active_member on the server ("west" -> "east"),
				// re-planning the unchanged config must be a no-op. Without the DiffSuppressFunc on
				// active_member, this step planned a destroy-and-recreate of the pair (and, via
				// parent_resource_crn, of any endpoint bound to it).
				PreConfig: func() {
					if err := triggerWiremockScenarioHook(mockServerUrl, switchoverPairFailoverHook); err != nil {
						t.Fatal(err)
					}
				},
				Config:             testAccCheckSwitchoverPairConfig(mockServerUrl, "prod-kafka-dr-v2"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// The refreshed state reflects the server-side failover without a diff on the inputs.
				Config: testAccCheckSwitchoverPairConfig(mockServerUrl, "prod-kafka-dr-v2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "active_member", "east"),
					resource.TestCheckResourceAttr(switchoverPairResourceLabel, "first_active", "west"),
				),
			},
			{
				// Documented behavior: once the pair exists, editing active_member in the configuration
				// has no effect (the diff is suppressed), so this must also be a no-op. Failovers are
				// driven by confluent_switchover_pair_failover, never by this attribute.
				Config:             testAccCheckSwitchoverPairConfigWithActiveMember(mockServerUrl, "prod-kafka-dr-v2", "east"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				// Immutable attributes: renaming a member on an existing pair must fail the plan (not
				// silently destroy and recreate the live DR pair, which is what ForceNew alone would do).
				Config:      testAccCheckSwitchoverPairConfigWithMemberName(mockServerUrl, "prod-kafka-dr-v2", "west-renamed"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`"members" cannot be changed after the switchover pair is created`),
			},
		},
	})
}

func testAccCheckSwitchoverPairConfig(mockServerUrl, displayName string) string {
	return testAccCheckSwitchoverPairConfigFull(mockServerUrl, displayName, "west", "west")
}

func testAccCheckSwitchoverPairConfigWithActiveMember(mockServerUrl, displayName, activeMember string) string {
	return testAccCheckSwitchoverPairConfigFull(mockServerUrl, displayName, activeMember, "west")
}

func testAccCheckSwitchoverPairConfigWithMemberName(mockServerUrl, displayName, westMemberName string) string {
	return testAccCheckSwitchoverPairConfigFull(mockServerUrl, displayName, "west", westMemberName)
}

func testAccCheckSwitchoverPairConfigFull(mockServerUrl, displayName, activeMember, westMemberName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	resource "confluent_switchover_pair" "main" {
		display_name    = "%s"
		active_member   = "%s"
		environment_crn = "%s"

		members {
			name       = "%s"
			member_crn = "%s"
		}

		members {
			name       = "east"
			member_crn = "%s"
		}
	}
	`, mockServerUrl, displayName, activeMember, switchoverPairEnvironmentCrn, westMemberName, switchoverPairWestMemberCrn, switchoverPairEastMemberCrn)
}

func TestAccDataSourceSwitchoverPair(t *testing.T) {
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

	createPairResponse, _ := os.ReadFile("../testdata/switchover/create_pair.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		WillReturn(
			string(createPairResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSwitchoverPairConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "id", "sw-abc123"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "display_name", "prod-kafka-dr"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "active_member", "west"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "first_active", "west"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "phase", "READY_TO_FAILOVER"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "members.#", "2"),
					resource.TestCheckResourceAttr(switchoverPairDataSourceLabel, "members.0.member_crn", switchoverPairWestMemberCrn),
				),
			},
		},
	})
}

func testAccCheckDataSourceSwitchoverPairConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	data "confluent_switchover_pair" "main" {
		id              = "sw-abc123"
		environment_crn = "%s"
	}
	`, mockServerUrl, switchoverPairEnvironmentCrn)
}
