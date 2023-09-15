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
	scenarioStateGcpPlaIsProvisioning          = "The new gcp private link access is provisioning"
	scenarioStateGcpPlaIsDeprovisioning        = "The new gcp private link access is deprovisioning"
	scenarioStateGcpPlaHasBeenCreated          = "The new gcp private link access has been just created"
	scenarioStateGcpPlaIsInDeprovisioningState = "The new gcp private link access is in deprovisioning state"
	scenarioStateGcpPlaHasBeenDeleted          = "The new gcp private link access's deletion has been just completed"
	gcpPlaScenarioName                         = "confluent_private_link_access Resource Lifecycle"
	gcpPlaEnvironmentId                        = "env-j5zwzm"
	gcpPlaNetworkId                            = "n-6ky22p"
	gcpPlaId                                   = "pla-gewd7g"
	gcpPlaProject                              = "test-project"
)

var gcpPlaUrlPath = fmt.Sprintf("/networking/v1/private-link-accesses/%s", gcpPlaId)

func TestAccGcpPrivateLinkAccess(t *testing.T) {
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
	createGcpPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/gcp/create_pla.json")
	createGcpPlaStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/private-link-accesses")).
		InScenario(gcpPlaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGcpPlaIsProvisioning).
		WillReturn(
			string(createGcpPlaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createGcpPlaStub)

	readProvisioningGcpPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/gcp/read_provisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPlaUrlPath)).
		InScenario(gcpPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPlaIsProvisioning).
		WillSetStateTo(scenarioStateGcpPlaHasBeenCreated).
		WillReturn(
			string(readProvisioningGcpPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedGcpPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/gcp/read_created_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPlaUrlPath)).
		InScenario(gcpPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPlaHasBeenCreated).
		WillReturn(
			string(readCreatedGcpPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteGcpPlaStub := wiremock.Delete(wiremock.URLPathEqualTo(gcpPlaUrlPath)).
		InScenario(gcpPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPlaHasBeenCreated).
		WillSetStateTo(scenarioStateGcpPlaIsInDeprovisioningState).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteGcpPlaStub)

	readDeprovisioningGcpPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/gcp/read_deprovisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(gcpPlaUrlPath)).
		InScenario(gcpPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPlaIsDeprovisioning).
		WillSetStateTo(scenarioStateGcpPlaHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningGcpPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedGcpPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/gcp/read_deleted_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPlaUrlPath)).
		InScenario(gcpPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPlaHasBeenDeleted).
		WillReturn(
			string(readDeletedGcpPlaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	gcpPlaDisplayName := "prod-pl-use3"
	gcpPlaResourceLabel := "test"
	fullGcpPlaResourceLabel := fmt.Sprintf("confluent_private_link_access.%s", gcpPlaResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGcpPlaDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGcpPlaConfig(mockServerUrl, gcpPlaDisplayName, gcpPlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpPlaExists(fullGcpPlaResourceLabel),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "id", gcpPlaId),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "display_name", gcpPlaDisplayName),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "gcp.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "gcp.0.project", gcpPlaProject),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "environment.0.id", gcpPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "network.0.id", gcpPlaNetworkId),
				),
			},
			{
				Config: testAccCheckGcpPlaConfigWithoutDisplayNameSet(mockServerUrl, gcpPlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpPlaExists(fullGcpPlaResourceLabel),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "id", gcpPlaId),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "display_name", gcpPlaDisplayName),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "gcp.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "gcp.0.project", gcpPlaProject),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "environment.0.id", gcpPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPlaResourceLabel, "network.0.id", gcpPlaNetworkId),
				),
			},
			{
				ResourceName:      fullGcpPlaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					gcpPlaId := resources[fullGcpPlaResourceLabel].Primary.ID
					environmentId := resources[fullGcpPlaResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + gcpPlaId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createGcpPlaStub, fmt.Sprintf("POST %s", gcpPlaUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteGcpPlaStub, fmt.Sprintf("DELETE %s?environment=%s", gcpPlaUrlPath, gcpPlaEnvironmentId), expectedCountOne)
}

func testAccCheckGcpPlaDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each private link access is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_private_link_access" {
			continue
		}
		deletedPrivateLinkAccessId := rs.Primary.ID
		req := c.netClient.PrivateLinkAccessesNetworkingV1Api.GetNetworkingV1PrivateLinkAccess(c.netApiContext(context.Background()), deletedPrivateLinkAccessId).Environment(gcpPlaEnvironmentId)
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

func testAccCheckGcpPlaConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
        display_name = "%s"
	    gcp {
		  project = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, gcpPlaProject, gcpPlaEnvironmentId, gcpPlaNetworkId)
}

func testAccCheckGcpPlaConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
	    gcp {
		  project = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, gcpPlaProject, gcpPlaEnvironmentId, gcpPlaNetworkId)
}

func testAccCheckGcpPlaExists(n string) resource.TestCheckFunc {
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
