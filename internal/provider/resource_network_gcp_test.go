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
	scenarioStateGcpNetworkIsProvisioning = "The new gcp network is in provisioning state"
	scenarioStateGcpNetworkHasBeenCreated = "The new gcp network has been just created"
	scenarioStateGcpNetworkHasBeenDeleted = "The new gcp network has been deleted"
	gcpNetworkScenarioName                = "confluent_network gcp Resource Lifecycle"
	gcpNetworkCloud                       = "GCP"
	gcpNetworkRegion                      = "us-central1"
	gcpNetworkConnectionType              = "PRIVATELINK"
	gcpNetworkEnvironmentId               = "env-j5zwzm"
	gcpNetworkId                          = "n-6ky22p"
	gcpDnsDomain                          = "6ky22p.us-central1.gcp.confluent.cloud"
	gcpNetworkResourceName                = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-j5zwzm/network=n-6ky22p"

	firstZoneGcpNetwork           = "us-central1-a"
	firstZoneSubdomainGcpNetwork  = "us-central1-a.6ky22p.us-central1.gcp.confluent.cloud"
	secondZoneGcpNetwork          = "us-central1-b"
	secondZoneSubdomainGcpNetwork = "us-central1-b.6ky22p.us-central1.gcp.confluent.cloud"
	thirdZoneGcpNetwork           = "us-central1-c"
	thirdZoneSubdomainGcpNetwork  = "us-central1-c.6ky22p.us-central1.gcp.confluent.cloud"

	firstGcpPlaAliasName   = "us-central1-a"
	firstGcpPlaAliasValue  = "projects/test-project/regions/us-central1/serviceAttachments/s-w2r4v-service-attachment-us-central1-a"
	secondGcpPlaAliasName  = "us-central1-b"
	secondGcpPlaAliasValue = "projects/test-project/regions/us-central1/serviceAttachments/s-w2r4v-service-attachment-us-central1-b"
	thirdGcpPlaAliasName   = "us-central1-c"
	thirdGcpPlaAliasValue  = "projects/test-project/regions/us-central1/serviceAttachments/s-w2r4v-service-attachment-us-central1-c"
)

var gcpNetworkUrlPath = fmt.Sprintf("/networking/v1/networks/%s", gcpNetworkId)

func TestAccGcpNetwork(t *testing.T) {
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
	createGcpNetworkResponse, _ := ioutil.ReadFile("../testdata/network/gcp/create_network.json")
	createGcpNetworkStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/networks")).
		InScenario(gcpNetworkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGcpNetworkIsProvisioning).
		WillReturn(
			string(createGcpNetworkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createGcpNetworkStub)

	readProvisioningGcpNetworkResponse, _ := ioutil.ReadFile("../testdata/network/gcp/read_provisioning_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpNetworkUrlPath)).
		InScenario(gcpNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpNetworkIsProvisioning).
		WillSetStateTo(scenarioStateGcpNetworkHasBeenCreated).
		WillReturn(
			string(readProvisioningGcpNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedGcpNetworkResponse, _ := ioutil.ReadFile("../testdata/network/gcp/read_created_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpNetworkUrlPath)).
		InScenario(gcpNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpNetworkHasBeenCreated).
		WillReturn(
			string(readCreatedGcpNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteGcpNetworkStub := wiremock.Delete(wiremock.URLPathEqualTo(gcpNetworkUrlPath)).
		InScenario(gcpNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpNetworkHasBeenCreated).
		WillSetStateTo(scenarioStateGcpNetworkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteGcpNetworkStub)

	readDeletedGcpNetworkResponse, _ := ioutil.ReadFile("../testdata/network/gcp/read_deleted_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpNetworkUrlPath)).
		InScenario(gcpNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpNetworkHasBeenDeleted).
		WillReturn(
			string(readDeletedGcpNetworkResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	gcpNetworkDisplayName := "s-nk99e"
	gcpNetworkResourceLabel := "test"
	fullGcpNetworkResourceLabel := fmt.Sprintf("confluent_network.%s", gcpNetworkResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGcpNetworkDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGcpNetworkConfig(mockServerUrl, gcpNetworkDisplayName, gcpNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpNetworkExists(fullGcpNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramId, gcpNetworkId),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramDisplayName, gcpNetworkDisplayName),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramCloud, gcpNetworkCloud),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), gcpNetworkConnectionType),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramRegion, gcpNetworkRegion),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), firstZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), secondZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), thirdZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramResourceName, gcpNetworkResourceName),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramDnsDomain, gcpDnsDomain),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneGcpNetwork), firstZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneGcpNetwork), secondZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneGcpNetwork), thirdZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, "gcp.0.private_service_connect_service_attachments.%", "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, firstGcpPlaAliasName), firstGcpPlaAliasValue),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, secondGcpPlaAliasName), secondGcpPlaAliasValue),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, thirdGcpPlaAliasName), thirdGcpPlaAliasValue),
				),
			},
			{
				Config: testAccCheckGcpNetworkConfigWithoutDisplayNameSet(mockServerUrl, gcpNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpNetworkExists(fullGcpNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramId, gcpNetworkId),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramDisplayName, gcpNetworkDisplayName),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramCloud, gcpNetworkCloud),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), gcpNetworkConnectionType),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), gcpNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramRegion, gcpNetworkRegion),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), firstZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), secondZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), thirdZoneGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramResourceName, gcpNetworkResourceName),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, paramDnsDomain, gcpDnsDomain),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneGcpNetwork), firstZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneGcpNetwork), secondZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneGcpNetwork), thirdZoneSubdomainGcpNetwork),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, "gcp.0.private_service_connect_service_attachments.%", "3"),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, firstGcpPlaAliasName), firstGcpPlaAliasValue),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, secondGcpPlaAliasName), secondGcpPlaAliasValue),
					resource.TestCheckResourceAttr(fullGcpNetworkResourceLabel, fmt.Sprintf("%s.0.%s.%s", paramGcp, paramPrivateServiceConnectServiceAttachments, thirdGcpPlaAliasName), thirdGcpPlaAliasValue),
				),
			},
			{
				ResourceName:      fullGcpNetworkResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					gcpNetworkId := resources[fullGcpNetworkResourceLabel].Primary.ID
					environmentId := resources[fullGcpNetworkResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + gcpNetworkId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createGcpNetworkStub, fmt.Sprintf("POST %s", gcpNetworkUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteGcpNetworkStub, fmt.Sprintf("DELETE %s?environment=%s", gcpNetworkUrlPath, gcpNetworkEnvironmentId), expectedCountOne)
}

func testAccCheckGcpNetworkDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each gcp network is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_network" {
			continue
		}
		deletedGcpNetworkId := rs.Primary.ID
		req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(context.Background()), deletedGcpNetworkId).Environment(gcpNetworkEnvironmentId)
		deletedGcpNetwork, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedGcpNetwork.Id != nil {
			// Otherwise return the error
			if *deletedGcpNetwork.Id == rs.Primary.ID {
				return fmt.Errorf("gcp network (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckGcpNetworkConfig(mockServerUrl, networkDisplayName, resourceLabel string) string {
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
	`, mockServerUrl, resourceLabel, networkDisplayName, gcpNetworkCloud, gcpNetworkRegion, gcpNetworkConnectionType, gcpNetworkEnvironmentId)
}

func testAccCheckGcpNetworkConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
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
	`, mockServerUrl, resourceLabel, gcpNetworkCloud, gcpNetworkRegion, gcpNetworkConnectionType, gcpNetworkEnvironmentId)
}

func testAccCheckGcpNetworkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s gcp network has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s gcp network", n)
		}

		return nil
	}
}
