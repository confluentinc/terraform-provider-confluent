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
	switchoverPairResourceLabel   = "confluent_switchover_pair.main"
	switchoverPairDataSourceLabel = "data.confluent_switchover_pair.main"
	switchoverPairsUrlPath        = "/switchover/v1/switchover-pairs"
	switchoverPairReadUrlPath     = "/switchover/v1/switchover-pairs/sw-abc123"
	switchoverPairScenarioName    = "confluent_switchover_pair Resource Lifecycle"

	scenarioStateSwitchoverPairHasBeenCreated = "The switchover pair has been created"
	scenarioStateSwitchoverPairHasBeenUpdated = "The switchover pair has been updated"
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

	createPairResponse, _ := os.ReadFile("../testdata/switchover/create_pair.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(switchoverPairsUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSwitchoverPairHasBeenCreated).
		WillReturn(
			string(createPairResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

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

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(switchoverPairReadUrlPath)).
		InScenario(switchoverPairScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverPairHasBeenUpdated).
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
		},
	})
}

func testAccCheckSwitchoverPairConfig(mockServerUrl, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	resource "confluent_switchover_pair" "main" {
		display_name    = "%s"
		active_member   = "west"
		environment_crn = "%s"

		members {
			name       = "west"
			member_crn = "%s"
		}

		members {
			name       = "east"
			member_crn = "%s"
		}
	}
	`, mockServerUrl, displayName, switchoverPairEnvironmentCrn, switchoverPairWestMemberCrn, switchoverPairEastMemberCrn)
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
