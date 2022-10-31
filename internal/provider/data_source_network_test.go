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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceNetworkScenarioName = "confluent_network Data Source Lifecycle"
	networkDataSourceLabel        = "example"
	azureNetworkDisplayName       = "s-nk99e"
	awsNetworkDisplayName         = "s-n9553"
)

var fullNetworkDataSourceLabel = fmt.Sprintf("data.confluent_network.%s", networkDataSourceLabel)

func TestAccDataSourceNetwork(t *testing.T) {
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

	readCreatedAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws/read_created_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsNetworkUrlPath)).
		InScenario(dataSourceNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsNetworkEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readNetworksResponse, _ := ioutil.ReadFile("../testdata/network/read_networks.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/networks")).
		InScenario(dataSourceNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azureNetworkEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readNetworksResponse),
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
				Config: testAccCheckDataSourceAwsNetworkConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsNetworkExists(fullNetworkDataSourceLabel),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramId, awsNetworkId),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramDisplayName, awsNetworkDisplayName),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramCloud, awsNetworkCloud),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), awsNetworkConnectionType),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), awsNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramRegion, awsNetworkRegion),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramZones), strconv.Itoa(len(awsNetworkZones))),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0", paramZones), awsNetworkZones[0]),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.1", paramZones), awsNetworkZones[1]),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.2", paramZones), awsNetworkZones[2]),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramResourceName, awsNetworkResourceName),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramDnsDomain, awsDnsDomain),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAwsNetwork), firstZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAwsNetwork), secondZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAwsNetwork), thirdZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramVpc), awsNetworkVpc),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramAccount), awsNetworkAccount),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramPrivateLinkEndpointService), awsNetworkPrivateLinkEndpointService),
				),
			},
			{
				Config: testAccCheckDataSourceAzureNetworkConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureNetworkExists(fullNetworkDataSourceLabel),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramId, azureNetworkId),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramDisplayName, azureNetworkDisplayName),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramCloud, azureNetworkCloud),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), azureNetworkConnectionType),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramRegion, azureNetworkRegion),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.#", paramZones), "3"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0", paramZones), "1"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.1", paramZones), "2"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.2", paramZones), "3"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramResourceName, azureNetworkResourceName),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, paramDnsDomain, azureDnsDomain),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAzureNetwork), firstZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAzureNetwork), secondZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAzureNetwork), thirdZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, "azure.0.private_link_service_aliases.%", "3"),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, firstPlaAliasName), firstPlaAliasValue),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, secondPlaAliasName), secondPlaAliasValue),
					resource.TestCheckResourceAttr(fullNetworkDataSourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, thirdPlaAliasName), thirdPlaAliasValue),
				),
			},
		},
	})
}

func testAccCheckDataSourceAzureNetworkConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_network" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, networkDataSourceLabel, azureNetworkDisplayName, azureNetworkEnvironmentId)
}

func testAccCheckDataSourceAwsNetworkConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_network" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, networkDataSourceLabel, awsNetworkId, awsNetworkEnvironmentId)
}
