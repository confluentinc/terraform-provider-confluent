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
	networkLinkEndpointDataSourceScenarioName = "confluent_network_link_endpoint Data Source Lifecycle"

	networkLinkEndpointReadUrlPath = "/networking/v1/network-link-endpoints/nle-6wvqx9"
	networkLinkEndpointId          = "nle-6wvqx9"
	networkLinkEndpointLabel       = "data.confluent_network_link_endpoint.nle"
)

func TestAccDataSourceNetworkLinkEndpoint(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readNetworkLinkEndpointResponse, _ := ioutil.ReadFile("../testdata/network_link_endpoint/read_nle.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
		InScenario(networkLinkEndpointDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readNetworkLinkEndpointResponse),
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
				Config: testAccCheckDataSourceNetworkLinkEndpointWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "id", "nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "resource_name", "crn://confluent.cloud/organization=foo/environment=env-d1o8qo/network=n-6kqnx2/network-link-endpoint=nle-6wvqx9"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "display_name", "network-link-endpoint-1"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "environment.0.id", "env-d1o8qo"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "network.0.id", "n-6kqnx2"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "network_link_service.#", "1"),
					resource.TestCheckResourceAttr(networkLinkEndpointLabel, "network_link_service.0.id", "nls-6942jn"),
				),
			},
		},
	})
}

func testAccCheckDataSourceNetworkLinkEndpointWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}

	data "confluent_network_link_endpoint" "nle" {
		id = "%s"
        environment {
			id = "env-d1o8qo"
	  	}
	}
	`, mockServerUrl, networkLinkEndpointId)
}
