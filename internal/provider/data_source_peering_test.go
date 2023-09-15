// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourcePeeringScenarioName = "confluent_peering Data Source Lifecycle"
	peeringDataSourceLabel        = "example"
	peeringDataSourceDisplayName  = "my-test-peering"
)

var fullPeeringDataSourceLabel = fmt.Sprintf("data.confluent_peering.%s", peeringDataSourceLabel)

func TestAccDataSourcePeering(t *testing.T) {
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

	readCreatedAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_created_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(dataSourcePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPeeringsResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_peerings.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/peerings")).
		InScenario(dataSourcePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPeeringsResponse),
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
				Config: testAccCheckDataSourcePeeringWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullPeeringDataSourceLabel),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "display_name", peeringDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.vpc", awsPeeringVpcId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.#", "2"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.0", awsPeeringRoutes[0]),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.1", awsPeeringRoutes[1]),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
			{
				Config: testAccCheckDataSourcePeeringWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullPeeringDataSourceLabel),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "display_name", peeringDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.vpc", awsPeeringVpcId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.#", "2"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.0", awsPeeringRoutes[0]),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.routes.1", awsPeeringRoutes[1]),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
		},
	})
}

func testAccCheckDataSourcePeeringWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_peering" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, peeringDataSourceLabel, peeringDataSourceDisplayName, awsPeeringEnvironmentId)
}

func testAccCheckDataSourcePeeringWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_peering" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, peeringDataSourceLabel, awsPeeringId, awsPeeringEnvironmentId)
}
