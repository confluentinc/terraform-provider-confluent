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
	scenarioStateGcpPeeringIsProvisioning   = "The new gcp peering is provisioning"
	scenarioStateGcpPeeringIsDeprovisioning = "The new gcp peering is deprovisioning"
	scenarioStateGcpPeeringHasBeenCreated   = "The new gcp peering has been just created"
	scenarioStateGcpPeeringHasBeenDeleted   = "The new gcp peering's deletion has been just completed"
	gcpPeeringScenarioName                  = "confluent_gcp Peering Gcp Resource Lifecycle"
	gcpPeeringEnvironmentId                 = "env-gz903"
	gcpPeeringNetworkId                     = "n-gez54g"
	gcpPeeringId                            = "peer-6me8yg"
	gcpProject                              = "superb-gear-123456"
	gcpVpcNetwork                           = "test-vpc"
)

var gcpPeeringUrlPath = fmt.Sprintf("/networking/v1/peerings/%s", gcpPeeringId)

func TestAccGcpPeeringAccess(t *testing.T) {
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
	createGcpPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/gcp/create_peering.json")
	createGcpPeeringStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/peerings")).
		InScenario(gcpPeeringScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGcpPeeringIsProvisioning).
		WillReturn(
			string(createGcpPeeringResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createGcpPeeringStub)

	readProvisioningGcpPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/gcp/read_provisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPeeringUrlPath)).
		InScenario(gcpPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPeeringIsProvisioning).
		WillSetStateTo(scenarioStateGcpPeeringHasBeenCreated).
		WillReturn(
			string(readProvisioningGcpPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedGcpPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/gcp/read_created_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPeeringUrlPath)).
		InScenario(gcpPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPeeringHasBeenCreated).
		WillReturn(
			string(readCreatedGcpPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteGcpPeeringStub := wiremock.Delete(wiremock.URLPathEqualTo(gcpPeeringUrlPath)).
		InScenario(gcpPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPeeringHasBeenCreated).
		WillSetStateTo(scenarioStateGcpPeeringIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteGcpPeeringStub)

	readDeprovisioningGcpPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/gcp/read_deprovisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(gcpPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPeeringIsDeprovisioning).
		WillSetStateTo(scenarioStateGcpPeeringHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningGcpPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedGcpPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/gcp/read_deleted_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(gcpPeeringUrlPath)).
		InScenario(gcpPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(gcpPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateGcpPeeringHasBeenDeleted).
		WillReturn(
			string(readDeletedGcpPeeringResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	gcpPeeringDisplayName := "my-test-peering"
	gcpPeeringResourceLabel := "test"
	fullGcpPeeringResourceLabel := fmt.Sprintf("confluent_peering.%s", gcpPeeringResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGcpPeeringDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGcpPeeringConfig(mockServerUrl, gcpPeeringDisplayName, gcpPeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpPeeringExists(fullGcpPeeringResourceLabel),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "id", gcpPeeringId),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "display_name", gcpPeeringDisplayName),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.project", gcpProject),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.vpc_network", gcpVpcNetwork),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.import_custom_routes", "false"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "environment.0.id", gcpPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "network.0.id", gcpPeeringNetworkId),
				),
			},
			{
				Config: testAccCheckGcpPeeringConfigWithoutDisplayNameSet(mockServerUrl, gcpPeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpPeeringExists(fullGcpPeeringResourceLabel),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "id", gcpPeeringId),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "display_name", gcpPeeringDisplayName),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.project", gcpProject),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.vpc_network", gcpVpcNetwork),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "gcp.0.import_custom_routes", "false"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "aws.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "environment.0.id", gcpPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullGcpPeeringResourceLabel, "network.0.id", gcpPeeringNetworkId),
				),
			},
			{
				ResourceName:      fullGcpPeeringResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					gcpPeeringId := resources[fullGcpPeeringResourceLabel].Primary.ID
					environmentId := resources[fullGcpPeeringResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + gcpPeeringId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createGcpPeeringStub, fmt.Sprintf("POST %s", gcpPeeringUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteGcpPeeringStub, fmt.Sprintf("DELETE %s?environment=%s", gcpPeeringUrlPath, gcpPeeringEnvironmentId), expectedCountOne)
}

func testAccCheckGcpPeeringDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each gcp peering is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_peering" {
			continue
		}
		deletedPeeringId := rs.Primary.ID
		req := c.netClient.PeeringsNetworkingV1Api.GetNetworkingV1Peering(c.netApiContext(context.Background()), deletedPeeringId).Environment(gcpPeeringEnvironmentId)
		deletedPeering, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedPeering.Id != nil {
			// Otherwise return the error
			if *deletedPeering.Id == rs.Primary.ID {
				return fmt.Errorf("gcp peering (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckGcpPeeringConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
        display_name = "%s"
	    gcp {
		  project = "%s"
          vpc_network = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, gcpProject, gcpVpcNetwork, gcpPeeringEnvironmentId, gcpPeeringNetworkId)
}

func testAccCheckGcpPeeringConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
	    gcp {
		  project = "%s"
          vpc_network = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, gcpProject, gcpVpcNetwork, gcpPeeringEnvironmentId, gcpPeeringNetworkId)
}

func testAccCheckGcpPeeringExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s AWS Peering has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Gcp Peering", n)
		}

		return nil
	}
}
