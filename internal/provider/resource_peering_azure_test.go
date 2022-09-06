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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateAzurePeeringIsProvisioning   = "The new azure peering is provisioning"
	scenarioStateAzurePeeringIsDeprovisioning = "The new azure peering is deprovisioning"
	scenarioStateAzurePeeringHasBeenCreated   = "The new azure peering has been just created"
	scenarioStateAzurePeeringHasBeenDeleted   = "The new azure peering's deletion has been just completed"
	azurePeeringScenarioName                  = "confluent_azure Peering Azure Resource Lifecycle"
	azurePeeringEnvironmentId                 = "env-gz903"
	azurePeeringNetworkId                     = "n-6k5026"
	azurePeeringId                            = "peer-g49jz6"
	azureTenant                               = "1111tttt-1111-1111-1111-111111tttttt"
	azurePeeringVNetResourceId                = "/subscriptions/1111ssss-1111-1111-1111-111111ssssss/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test"
	azurePeeringCustomerRegion                = "eastus"
)

var azurePeeringUrlPath = fmt.Sprintf("/networking/v1/peerings/%s", azurePeeringId)

func TestAccAzurePeeringAccess(t *testing.T) {
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
	createAzurePeeringResponse, _ := ioutil.ReadFile("../testdata/peering/azure/create_peering.json")
	createAzurePeeringStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/peerings")).
		InScenario(azurePeeringScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAzurePeeringIsProvisioning).
		WillReturn(
			string(createAzurePeeringResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAzurePeeringStub)

	readProvisioningAzurePeeringResponse, _ := ioutil.ReadFile("../testdata/peering/azure/read_provisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePeeringUrlPath)).
		InScenario(azurePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePeeringIsProvisioning).
		WillSetStateTo(scenarioStateAzurePeeringHasBeenCreated).
		WillReturn(
			string(readProvisioningAzurePeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAzurePeeringResponse, _ := ioutil.ReadFile("../testdata/peering/azure/read_created_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePeeringUrlPath)).
		InScenario(azurePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePeeringHasBeenCreated).
		WillReturn(
			string(readCreatedAzurePeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAzurePeeringStub := wiremock.Delete(wiremock.URLPathEqualTo(azurePeeringUrlPath)).
		InScenario(azurePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePeeringHasBeenCreated).
		WillSetStateTo(scenarioStateAzurePeeringIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAzurePeeringStub)

	readDeprovisioningAzurePeeringResponse, _ := ioutil.ReadFile("../testdata/peering/azure/read_deprovisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(azurePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePeeringIsDeprovisioning).
		WillSetStateTo(scenarioStateAzurePeeringHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAzurePeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAzurePeeringResponse, _ := ioutil.ReadFile("../testdata/peering/azure/read_deleted_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePeeringUrlPath)).
		InScenario(azurePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePeeringHasBeenDeleted).
		WillReturn(
			string(readDeletedAzurePeeringResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	azurePeeringDisplayName := "my-test-peering"
	azurePeeringResourceLabel := "test"
	fullAzurePeeringResourceLabel := fmt.Sprintf("confluent_peering.%s", azurePeeringResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAzurePeeringDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAzurePeeringConfig(mockServerUrl, azurePeeringDisplayName, azurePeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzurePeeringExists(fullAzurePeeringResourceLabel),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "id", azurePeeringId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "display_name", azurePeeringDisplayName),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.tenant", azureTenant),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.vnet", azurePeeringVNetResourceId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.customer_region", azurePeeringCustomerRegion),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "environment.0.id", azurePeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "network.0.id", azurePeeringNetworkId),
				),
			},
			{
				Config: testAccCheckAzurePeeringConfigWithoutDisplayNameSet(mockServerUrl, azurePeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzurePeeringExists(fullAzurePeeringResourceLabel),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "id", azurePeeringId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "display_name", azurePeeringDisplayName),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.tenant", azureTenant),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.vnet", azurePeeringVNetResourceId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "azure.0.customer_region", azurePeeringCustomerRegion),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "environment.0.id", azurePeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePeeringResourceLabel, "network.0.id", azurePeeringNetworkId),
				),
			},
			{
				ResourceName:      fullAzurePeeringResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					azurePeeringId := resources[fullAzurePeeringResourceLabel].Primary.ID
					environmentId := resources[fullAzurePeeringResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + azurePeeringId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAzurePeeringStub, fmt.Sprintf("POST %s", azurePeeringUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAzurePeeringStub, fmt.Sprintf("DELETE %s?environment=%s", azurePeeringUrlPath, azurePeeringEnvironmentId), expectedCountOne)
}

func testAccCheckAzurePeeringDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each azure peering is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_peering" {
			continue
		}
		deletedPeeringId := rs.Primary.ID
		req := c.netClient.PeeringsNetworkingV1Api.GetNetworkingV1Peering(c.netApiContext(context.Background()), deletedPeeringId).Environment(azurePeeringEnvironmentId)
		deletedPeering, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedPeering.Id != nil {
			// Otherwise return the error
			if *deletedPeering.Id == rs.Primary.ID {
				return fmt.Errorf("azure peering (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAzurePeeringConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
        display_name = "%s"
	    azure {
		  tenant = "%s"
          vnet = "%s"
          customer_region = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, azureTenant, azurePeeringVNetResourceId, azurePeeringCustomerRegion, azurePeeringEnvironmentId, azurePeeringNetworkId)
}

func testAccCheckAzurePeeringConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
	    azure {
		  tenant = "%s"
          vnet = "%s"
          customer_region = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, azureTenant, azurePeeringVNetResourceId, azurePeeringCustomerRegion, azurePeeringEnvironmentId, azurePeeringNetworkId)
}

func testAccCheckAzurePeeringExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s AWS Peering has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Azure Peering", n)
		}

		return nil
	}
}
