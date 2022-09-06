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
	scenarioStateAzurePlaIsProvisioning          = "The new azure private link access is provisioning"
	scenarioStateAzurePlaIsDeprovisioning        = "The new azure private link access is deprovisioning"
	scenarioStateAzurePlaHasBeenCreated          = "The new azure private link access has been just created"
	scenarioStateAzurePlaIsInDeprovisioningState = "The new azure private link access is in deprovisioning state"
	scenarioStateAzurePlaHasBeenDeleted          = "The new azure private link access's deletion has been just completed"
	azurePlaScenarioName                         = "confluent_private_link_access Resource Lifecycle"
	azurePlaEnvironmentId                        = "env-gz903"
	azurePlaNetworkId                            = "n-p8xo76"
	azurePlaId                                   = "pla-gz8rlg"
	azureSubscription                            = "1234abcd-12ab-34cd-1234-123456abcdef"
)

var azurePlaUrlPath = fmt.Sprintf("/networking/v1/private-link-accesses/%s", azurePlaId)

func TestAccAzurePrivateLinkAccess(t *testing.T) {
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
	createAzurePlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/azure/create_pla.json")
	createAzurePlaStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/private-link-accesses")).
		InScenario(azurePlaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAzurePlaIsProvisioning).
		WillReturn(
			string(createAzurePlaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAzurePlaStub)

	readProvisioningAzurePlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/azure/read_provisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePlaUrlPath)).
		InScenario(azurePlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePlaIsProvisioning).
		WillSetStateTo(scenarioStateAzurePlaHasBeenCreated).
		WillReturn(
			string(readProvisioningAzurePlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAzurePlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/azure/read_created_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePlaUrlPath)).
		InScenario(azurePlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePlaHasBeenCreated).
		WillReturn(
			string(readCreatedAzurePlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAzurePlaStub := wiremock.Delete(wiremock.URLPathEqualTo(azurePlaUrlPath)).
		InScenario(azurePlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePlaHasBeenCreated).
		WillSetStateTo(scenarioStateAzurePlaIsInDeprovisioningState).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAzurePlaStub)

	readDeprovisioningAzurePlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/azure/read_deprovisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(azurePlaUrlPath)).
		InScenario(azurePlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePlaIsDeprovisioning).
		WillSetStateTo(scenarioStateAzurePlaHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAzurePlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAzurePlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/azure/read_deleted_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(azurePlaUrlPath)).
		InScenario(azurePlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(azurePlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAzurePlaHasBeenDeleted).
		WillReturn(
			string(readDeletedAzurePlaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	azurePlaDisplayName := "prod-pl-use3"
	azurePlaResourceLabel := "test"
	fullAzurePlaResourceLabel := fmt.Sprintf("confluent_private_link_access.%s", azurePlaResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAzurePlaDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAzurePlaConfig(mockServerUrl, azurePlaDisplayName, azurePlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzurePlaExists(fullAzurePlaResourceLabel),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "id", azurePlaId),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "display_name", azurePlaDisplayName),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "azure.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "azure.0.subscription", azureSubscription),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "environment.0.id", azurePlaEnvironmentId),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "network.0.id", azurePlaNetworkId),
				),
			},
			{
				Config: testAccCheckAzurePlaConfigWithoutDisplayNameSet(mockServerUrl, azurePlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzurePlaExists(fullAzurePlaResourceLabel),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "id", azurePlaId),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "display_name", azurePlaDisplayName),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "azure.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "azure.0.subscription", azureSubscription),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "environment.0.id", azurePlaEnvironmentId),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAzurePlaResourceLabel, "network.0.id", azurePlaNetworkId),
				),
			},
			{
				ResourceName:      fullAzurePlaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					azurePlaId := resources[fullAzurePlaResourceLabel].Primary.ID
					environmentId := resources[fullAzurePlaResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + azurePlaId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAzurePlaStub, fmt.Sprintf("POST %s", azurePlaUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAzurePlaStub, fmt.Sprintf("DELETE %s?environment=%s", azurePlaUrlPath, azurePlaEnvironmentId), expectedCountOne)
}

func testAccCheckAzurePlaDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each private link access is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_private_link_access" {
			continue
		}
		deletedPrivateLinkAccessId := rs.Primary.ID
		req := c.netClient.PrivateLinkAccessesNetworkingV1Api.GetNetworkingV1PrivateLinkAccess(c.netApiContext(context.Background()), deletedPrivateLinkAccessId).Environment(azurePlaEnvironmentId)
		deletedPrivateLinkAccess, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedPrivateLinkAccess.Id != nil {
			// Otherwise return the error
			if *deletedPrivateLinkAccess.Id == rs.Primary.ID {
				return fmt.Errorf("private link access (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAzurePlaConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
        display_name = "%s"
	    azure {
		  subscription = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, azureSubscription, azurePlaEnvironmentId, azurePlaNetworkId)
}

func testAccCheckAzurePlaConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
	    azure {
		  subscription = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, azureSubscription, azurePlaEnvironmentId, azurePlaNetworkId)
}

func testAccCheckAzurePlaExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s Private Link Access has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Private Link Access", n)
		}

		return nil
	}
}
