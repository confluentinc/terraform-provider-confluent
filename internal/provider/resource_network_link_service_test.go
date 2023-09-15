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
	networkLinkServiceResourceScenarioName        = "confluent_network_link_service Resource Lifecycle"
	scenarioStateNetworkLinkServiceHasBeenCreated = "A new network link service has been just created"
	scenarioStateNetworkLinkServiceHasBeenUpdated = "A new network link service has been just updated"

	networkLinkServiceUrlPath       = "/networking/v1/network-link-services"
	networkLinkServiceResourceLabel = "confluent_network_link_service.nls"
)

func TestAccNetworkLinkService(t *testing.T) {
	mockServerUrl := tc.wiremockUrl
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createNLSResponse, _ := ioutil.ReadFile("../testdata/network_link_service/create_nls.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(networkLinkServiceUrlPath)).
		InScenario(networkLinkServiceResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateNetworkLinkServiceHasBeenCreated).
		WillReturn(
			string(createNLSResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readNLSResponse, _ := ioutil.ReadFile("../testdata/network_link_service/read_nls.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkServiceReadUrlPath)).
		InScenario(networkLinkServiceResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkServiceHasBeenCreated).
		WillReturn(
			string(readNLSResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedNLSResponse, _ := ioutil.ReadFile("../testdata/network_link_service/updated_nls.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(networkLinkServiceReadUrlPath)).
		InScenario(networkLinkServiceResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkServiceHasBeenCreated).
		WillSetStateTo(scenarioStateNetworkLinkServiceHasBeenUpdated).
		WillReturn(
			string(updatedNLSResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkServiceReadUrlPath)).
		InScenario(networkLinkServiceResourceScenarioName).
		WhenScenarioStateIs(scenarioStateNetworkLinkServiceHasBeenUpdated).
		WillReturn(
			string(updatedNLSResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(networkLinkServiceReadUrlPath)).
		InScenario(networkLinkServiceResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceNetworkLinkServiceWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "id", "nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-service=nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "display_name", "network-link-service-2"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.environments.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.environments.0", "env-nkv0pz"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.networks.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.networks.0", "n-6xr90w"),
				),
			},
			{
				Config: testAccCheckResourceUpdateNetworkLinkServiceWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "id", "nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-service=nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "display_name", "network-link-service-2"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.environments.#", "2"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.environments.0", "env-nkv0pz"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.environments.1", "env-nkvqqq"),
					resource.TestCheckResourceAttr(networkLinkServiceResourceLabel, "accept.0.networks.#", "0"),
				),
			},
		},
	})
}

func testAccCheckResourceNetworkLinkServiceWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource "confluent_network_link_service" "nls" {
        environment {
            id = "env-d1o8qo"
        }
        network {
            id = "n-6kqnx2"
        }
        display_name = "network-link-service-2"
        description = "Test NL service"
        accept {
            environments = ["env-nkv0pz"]
            networks = ["n-6xr90w"]
        }
    }
	`, mockServerUrl)
}

func testAccCheckResourceUpdateNetworkLinkServiceWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource "confluent_network_link_service" "nls" {
        environment {
            id = "env-d1o8qo"
        }
        network {
            id = "n-6kqnx2"
        }
        display_name = "network-link-service-2"
        description = "Test NL service"
        accept {
            environments = ["env-nkv0pz", "env-nkvqqq"]
            networks = []
        }
    }
	`, mockServerUrl)
}
