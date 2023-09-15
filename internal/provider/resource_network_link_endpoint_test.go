// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	networkLinkEndpointResourceScenarioName        = "confluent_network_link_endpoint Resource Lifecycle"
	scenarioStateNetworkLinkEndpointHasBeenCreated = "A new network link endpoint has been just created"
	scenarioStateNetworkLinkEndpointHasBeenUpdated = "A new network link endpoint has been just updated"
	scenarioStateNetworkLinkEndpointHasBeenDeleted = "A new network link endpoint has been just deleted"

	networkLinkEndpointUrlPath       = "/networking/v1/network-link-endpoints"
	networkLinkEndpointResourceLabel = "confluent_network_link_endpoint.nle"
)

func TestAccNetworkLinkEndpoint(t *testing.T) {
	mockServerUrl := tc.wiremockUrl
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createNLEResponse, _ := ioutil.ReadFile("../testdata/network_link_endpoint/read_nle.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(networkLinkEndpointUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateNetworkLinkEndpointHasBeenCreated).
		WillReturn(
			string(createNLEResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readNLEResponse, _ := ioutil.ReadFile("../testdata/network_link_endpoint/read_nle.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkEndpointHasBeenCreated).
		WillReturn(
			string(readNLEResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedNLEResponse, _ := ioutil.ReadFile("../testdata/network_link_endpoint/updated_nle.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkEndpointHasBeenCreated).
		WillSetStateTo(scenarioStateNetworkLinkEndpointHasBeenUpdated).
		WillReturn(
			string(updatedNLEResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkEndpointHasBeenUpdated).
		WillReturn(
			string(updatedNLEResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WillSetStateTo(scenarioStateNetworkLinkEndpointHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkEndpointHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceNetworkLinkEndpointWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "id", "nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-endpoint=nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "display_name", "network-link-endpoint-1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network_link_service.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network_link_service.0.id", "nls-6942jn"),
				),
			},
			{
				Config: testAccCheckResourceUpdateNetworkLinkEndpointWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "id", "nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-endpoint=nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "display_name", "network-link-endpoint-1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "description", "Updated test NL endpoint"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network_link_service.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointResourceLabel, "network_link_service.0.id", "nls-6942jn"),
				),
			},
		},
	})
}

func testAccCheckResourceNetworkLinkEndpointWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}

	resource "confluent_network_link_endpoint" "nle" {
        environment {
			id = "env-d1o8qo"
	  	}
        network {
            id = "n-6kqnx2"
        }
		display_name = "network-link-endpoint-1"
		description = "Test NL endpoint"
        network_link_service {
			id = "nls-6942jn"
		}
	}
	`, mockServerUrl)
}

func testAccCheckResourceUpdateNetworkLinkEndpointWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}

	resource "confluent_network_link_endpoint" "nle" {
		environment {
			id = "env-d1o8qo"
	  	}
        network {
            id = "n-6kqnx2"
        }
		display_name = "network-link-endpoint-1"
		description = "Updated test NL endpoint"
        network_link_service {
			id = "nls-6942jn"
		}
	}
	`, mockServerUrl)
}
