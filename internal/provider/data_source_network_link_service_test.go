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
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	networkLinkServiceDataSourceScenarioName = "confluent_network_link_service Data Source Lifecycle"

	networkLinkServiceReadUrlPath = "/networking/v1/network-link-services/nls-p2k0l1"
	networkLinkServiceListUrlPath = "/networking/v1/network-link-services"
	networkLinkServiceId          = "nls-p2k0l1"
	networkLinkServiceLabel       = "data.confluent_network_link_service.nls"
	networkLinkServiceDisplayName = "network-link-service-2"
)

func TestAccDataSourceNetworkLinkService(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	readNetworkLinkServiceResponse, _ := ioutil.ReadFile("../testdata/network_link_service/read_nls.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkServiceReadUrlPath)).
		InScenario(networkLinkServiceDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readNetworkLinkServiceResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	listNetworkLinkServiceResponse, _ := ioutil.ReadFile("../testdata/network_link_service/list_nls.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkServiceListUrlPath)).
		InScenario(networkLinkServiceDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(listNetworkLinkServiceResponse),
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
				Config: testAccCheckDataSourceNetworkLinkServiceWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "id", "nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-service=nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "display_name", "network-link-service-2"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.environments.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.environments.0", "env-nkv0pz"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.networks.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.networks.0", "n-6xr90w"),
				),
			},
			{
				Config: testAccCheckDataSourceNetworkLinkServiceWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "id", "nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-service=nls-p2k0l1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "display_name", "network-link-service-2"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.environments.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.environments.0", "env-nkv0pz"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.networks.#", "1"),
					resource.TestCheckResourceAttr(networkLinkServiceLabel, "accept.0.networks.0", "n-6xr90w"),
				),
			},
		},
	})
	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})
}

func testAccCheckDataSourceNetworkLinkServiceWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}

	data "confluent_network_link_service" "nls" {
		id = "%s"
        environment {
			id = "env-d1o8qo"
	  	}
	}
	`, mockServerUrl, networkLinkServiceId)
}

func testAccCheckDataSourceNetworkLinkServiceWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}

	data "confluent_network_link_service" "nls" {
		display_name = "%s"
        environment {
			id = "env-d1o8qo"
	  	}
	}
	`, mockServerUrl, networkLinkServiceDisplayName)
}
