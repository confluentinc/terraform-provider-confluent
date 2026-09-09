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
	switchoverEndpointResourceLabel   = "confluent_switchover_endpoint.main"
	switchoverEndpointDataSourceLabel = "data.confluent_switchover_endpoint.main"
	switchoverEndpointsUrlPath        = "/switchover/v1/switchover-endpoints"
	switchoverEndpointReadUrlPath     = "/switchover/v1/switchover-endpoints/se-abc123"
	switchoverEndpointScenarioName    = "confluent_switchover_endpoint Resource Lifecycle"

	scenarioStateSwitchoverEndpointHasBeenCreated = "The switchover endpoint has been created"
	scenarioStateSwitchoverEndpointHasBeenUpdated = "The switchover endpoint has been updated"
	scenarioStateSwitchoverEndpointHasBeenDeleted = "The switchover endpoint has been deleted"

	switchoverEndpointParentResourceCrn = "crn://confluent.cloud/organization=org-abc/environment=env-abc123/switchover-pair=sw-abc123"
	switchoverEndpointWestNetworkCrn    = "crn://confluent.cloud/organization=org-abc/environment=env-abc123/network=n-west01"
	switchoverEndpointEastNetworkCrn    = "crn://confluent.cloud/organization=org-abc/environment=env-def456/network=n-east01"
)

func TestAccSwitchoverEndpoint(t *testing.T) {
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

	createEndpointResponse, _ := os.ReadFile("../testdata/switchover/create_endpoint.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(switchoverEndpointsUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSwitchoverEndpointHasBeenCreated).
		WillReturn(
			string(createEndpointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverEndpointHasBeenCreated).
		WillReturn(
			string(createEndpointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedEndpointResponse, _ := os.ReadFile("../testdata/switchover/updated_endpoint.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverEndpointHasBeenCreated).
		WillSetStateTo(scenarioStateSwitchoverEndpointHasBeenUpdated).
		WillReturn(
			string(updatedEndpointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverEndpointHasBeenUpdated).
		WillReturn(
			string(updatedEndpointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverEndpointHasBeenUpdated).
		WillSetStateTo(scenarioStateSwitchoverEndpointHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeletedEndpointResponse, _ := os.ReadFile("../testdata/switchover/read_deleted_endpoint.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		InScenario(switchoverEndpointScenarioName).
		WhenScenarioStateIs(scenarioStateSwitchoverEndpointHasBeenDeleted).
		WillReturn(
			string(readDeletedEndpointResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSwitchoverEndpointConfig(mockServerUrl, "prod-kafka-dr-endpoint"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "id", "se-abc123"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "display_name", "prod-kafka-dr-endpoint"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "parent_resource_crn", switchoverEndpointParentResourceCrn),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "target", "west-platt"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "phase", "READY"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.#", "2"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.name", "west-platt"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.cloud", "AWS"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.connection_type", "PRIVATELINK"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.endpoint_filter.0.type", "private"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "endpoints.0.endpoint_filter.0.network_crn", switchoverEndpointWestNetworkCrn),
				),
			},
			{
				Config: testAccCheckSwitchoverEndpointConfig(mockServerUrl, "prod-kafka-dr-endpoint-v2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "id", "se-abc123"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "display_name", "prod-kafka-dr-endpoint-v2"),
					resource.TestCheckResourceAttr(switchoverEndpointResourceLabel, "parent_resource_crn", switchoverEndpointParentResourceCrn),
				),
			},
		},
	})
}

func testAccCheckSwitchoverEndpointConfig(mockServerUrl, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	resource "confluent_switchover_endpoint" "main" {
		display_name        = "%s"
		parent_resource_crn = "%s"

		endpoints {
			name = "west-platt"
			endpoint_filter {
				type        = "private"
				network_crn = "%s"
			}
		}

		endpoints {
			name = "east-platt"
			endpoint_filter {
				type        = "private"
				network_crn = "%s"
			}
		}
	}
	`, mockServerUrl, displayName, switchoverEndpointParentResourceCrn, switchoverEndpointWestNetworkCrn, switchoverEndpointEastNetworkCrn)
}

func TestAccDataSourceSwitchoverEndpoint(t *testing.T) {
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

	createEndpointResponse, _ := os.ReadFile("../testdata/switchover/create_endpoint.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(switchoverEndpointReadUrlPath)).
		WillReturn(
			string(createEndpointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSwitchoverEndpointConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "id", "se-abc123"),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "display_name", "prod-kafka-dr-endpoint"),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "parent_resource_crn", switchoverEndpointParentResourceCrn),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "target", "west-platt"),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "phase", "READY"),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "endpoints.#", "2"),
					resource.TestCheckResourceAttr(switchoverEndpointDataSourceLabel, "endpoints.0.connection_type", "PRIVATELINK"),
				),
			},
		},
	})
}

func testAccCheckDataSourceSwitchoverEndpointConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	data "confluent_switchover_endpoint" "main" {
		id              = "se-abc123"
		environment_crn = "%s"
	}
	`, mockServerUrl, switchoverPairEnvironmentCrn)
}
