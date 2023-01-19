// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateAzureNetworkIsProvisioning = "The new azure network is in provisioning state"
	scenarioStateAzureNetworkHasBeenCreated = "The new azure network has been just created"
	scenarioStateAzureNetworkHasBeenDeleted = "The new azure network has been deleted"
	azureNetworkScenarioName                = "confluent_network azure Resource Lifecycle"
	azureNetworkCloud                       = "AZURE"
	azureNetworkRegion                      = "centralus"
	azureNetworkConnectionType              = "PRIVATELINK"
	azureNetworkEnvironmentId               = "env-gz903"
	azureNetworkId                          = "n-p8xo76"
	azureDnsDomain                          = "p8xo76.centralus.azure.confluent.cloud"
	azureNetworkResourceName                = "crn://confluent.cloud/organization=foo/environment=env-gz903/network=n-p8xo76"

	firstZoneAzureNetwork           = "1"
	firstZoneSubdomainAzureNetwork  = "az1.p8xo76.centralus.azure.confluent.cloud"
	secondZoneAzureNetwork          = "2"
	secondZoneSubdomainAzureNetwork = "az2.p8xo76.centralus.azure.confluent.cloud"
	thirdZoneAzureNetwork           = "3"
	thirdZoneSubdomainAzureNetwork  = "az3.p8xo76.centralus.azure.confluent.cloud"

	firstPlaAliasName   = "1"
	firstPlaAliasValue  = "s-nk99e-privatelink-1.8c43dcd0-695c-1234-bc35-11fe6abb303a.centralus.azure.privatelinkservice"
	secondPlaAliasName  = "2"
	secondPlaAliasValue = "s-nk99e-privatelink-2.e4519a80-fcf9-1234-9163-167aa681e792.centralus.azure.privatelinkservice"
	thirdPlaAliasName   = "3"
	thirdPlaAliasValue  = "s-nk99e-privatelink-3.cb77af9e-3db1-1234-bf18-0f8dfba7662b.centralus.azure.privatelinkservice"
)

var azureNetworkUrlPath = fmt.Sprintf("/networking/v1/networks/%s", azureNetworkId)

func TestAccAzureNetwork(t *testing.T) {
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
	createAzureNetworkResponse, _ := ioutil.ReadFile("../testdata/network/azure/create_network.json")
	createAzureNetworkStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/networks")).
		InScenario(azureNetworkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAzureNetworkIsProvisioning).
		WillReturn(
			string(createAzureNetworkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAzureNetworkStub)

	readProvisioningAzureNetworkResponse, _ := ioutil.ReadFile("../testdata/network/azure/read_provisioning_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azureNetworkUrlPath)).
		InScenario(azureNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azureNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzureNetworkIsProvisioning).
		WillSetStateTo(scenarioStateAzureNetworkHasBeenCreated).
		WillReturn(
			string(readProvisioningAzureNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAzureNetworkResponse, _ := ioutil.ReadFile("../testdata/network/azure/read_created_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azureNetworkUrlPath)).
		InScenario(azureNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azureNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzureNetworkHasBeenCreated).
		WillReturn(
			string(readCreatedAzureNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAzureNetworkStub := wiremock.Delete(wiremock.URLPathEqualTo(azureNetworkUrlPath)).
		InScenario(azureNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azureNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzureNetworkHasBeenCreated).
		WillSetStateTo(scenarioStateAzureNetworkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAzureNetworkStub)

	readDeletedAzureNetworkResponse, _ := ioutil.ReadFile("../testdata/network/azure/read_deleted_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azureNetworkUrlPath)).
		InScenario(azureNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azureNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzureNetworkHasBeenDeleted).
		WillReturn(
			string(readDeletedAzureNetworkResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	azureNetworkDisplayName := "s-nk99e"
	azureNetworkResourceLabel := "test"
	fullAzureNetworkResourceLabel := fmt.Sprintf("confluent_network.%s", azureNetworkResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAzureNetworkDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAzureNetworkConfig(mockServerUrl, azureNetworkDisplayName, azureNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureNetworkExists(fullAzureNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramId, azureNetworkId),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramDisplayName, azureNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramCloud, azureNetworkCloud),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), azureNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramRegion, azureNetworkRegion),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), firstZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), secondZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), thirdZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramDnsConfig), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramDnsConfig, paramResolution), ""),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramResourceName, azureNetworkResourceName),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramDnsDomain, azureDnsDomain),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAzureNetwork), firstZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAzureNetwork), secondZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAzureNetwork), thirdZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, "azure.0.private_link_service_aliases.%", "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, firstPlaAliasName), firstPlaAliasValue),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, secondPlaAliasName), secondPlaAliasValue),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, thirdPlaAliasName), thirdPlaAliasValue),
				),
			},
			{
				Config: testAccCheckAzureNetworkConfigWithoutDisplayNameSet(mockServerUrl, azureNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureNetworkExists(fullAzureNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramId, azureNetworkId),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramDisplayName, azureNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramCloud, azureNetworkCloud),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), azureNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), azureNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramRegion, azureNetworkRegion),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), firstZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), secondZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), thirdZoneAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.#", paramDnsConfig), "1"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramDnsConfig, paramResolution), ""),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramResourceName, azureNetworkResourceName),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, paramDnsDomain, azureDnsDomain),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAzureNetwork), firstZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAzureNetwork), secondZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAzureNetwork), thirdZoneSubdomainAzureNetwork),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, "azure.0.private_link_service_aliases.%", "3"),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, firstPlaAliasName), firstPlaAliasValue),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, secondPlaAliasName), secondPlaAliasValue),
					resource.TestCheckResourceAttr(fullAzureNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramAzure, paramPrivateLinkServiceAliases, thirdPlaAliasName), thirdPlaAliasValue),
				),
			},
			{
				ResourceName:      fullAzureNetworkResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					azureNetworkId := resources[fullAzureNetworkResourceLabel].Primary.ID
					environmentId := resources[fullAzureNetworkResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + azureNetworkId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAzureNetworkStub, fmt.Sprintf("POST %s", azureNetworkUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAzureNetworkStub, fmt.Sprintf("DELETE %s?environment=%s", azureNetworkUrlPath, azureNetworkEnvironmentId), expectedCountOne)
}

func testAccCheckAzureNetworkDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each azure network is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_network" {
			continue
		}
		deletedAzureNetworkId := rs.Primary.ID
		req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(context.Background()), deletedAzureNetworkId).Environment(azureNetworkEnvironmentId)
		deletedAzureNetwork, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedAzureNetwork.Id != nil {
			// Otherwise return the error
			if *deletedAzureNetwork.Id == rs.Primary.ID {
				return fmt.Errorf("azure network (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAzureNetworkConfig(mockServerUrl, networkDisplayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_network" "%s" {
        display_name     = "%s"
	    cloud            = "%s"
	    region           = "%s"
	    connection_types = ["%s"]
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, networkDisplayName, azureNetworkCloud, azureNetworkRegion, azureNetworkConnectionType, azureNetworkEnvironmentId)
}

func testAccCheckAzureNetworkConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_network" "%s" {
	    cloud            = "%s"
	    region           = "%s"
	    connection_types = ["%s"]
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, azureNetworkCloud, azureNetworkRegion, azureNetworkConnectionType, azureNetworkEnvironmentId)
}

func testAccCheckAzureNetworkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s azure network has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s azure network", n)
		}

		return nil
	}
}
