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
	scenarioStateAwsPeeringIsProvisioning   = "The new aws peering is provisioning"
	scenarioStateAwsPeeringIsDeprovisioning = "The new aws peering is deprovisioning"
	scenarioStateAwsPeeringHasBeenCreated   = "The new aws peering has been just created"
	scenarioStateAwsPeeringHasBeenDeleted   = "The new aws peering's deletion has been just completed"
	awsPeeringScenarioName                  = "confluent_awsPeering AWS Resource Lifecycle"
	awsPeeringEnvironmentId                 = "env-gz903"
	awsPeeringNetworkId                     = "n-6k5026"
	awsPeeringId                            = "peer-gez27g"
	awsPeeringVpcId                         = "vpc-090dcc71c69483dc1"
	awsPeeringCustomerRegion                = "us-west-2"
)

var awsPeeringRoutes = []string{
	"10.0.0.0/24",
	"10.0.0.7/32",
}

var awsPeeringUrlPath = fmt.Sprintf("/networking/v1/peerings/%s", awsPeeringId)

func TestAccAwsPeeringAccess(t *testing.T) {
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
	createAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/create_peering.json")
	createAwsPeeringStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/peerings")).
		InScenario(awsPeeringScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAwsPeeringIsProvisioning).
		WillReturn(
			string(createAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAwsPeeringStub)

	readProvisioningAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_provisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(awsPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPeeringIsProvisioning).
		WillSetStateTo(scenarioStateAwsPeeringHasBeenCreated).
		WillReturn(
			string(readProvisioningAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_created_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(awsPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPeeringHasBeenCreated).
		WillReturn(
			string(readCreatedAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAwsPeeringStub := wiremock.Delete(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(awsPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPeeringHasBeenCreated).
		WillSetStateTo(scenarioStateAwsPeeringIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAwsPeeringStub)

	readDeprovisioningAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_deprovisioning_peering.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(awsPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPeeringIsDeprovisioning).
		WillSetStateTo(scenarioStateAwsPeeringHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_deleted_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(awsPeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPeeringHasBeenDeleted).
		WillReturn(
			string(readDeletedAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	awsPeeringDisplayName := "my-test-peering"
	awsPeeringResourceLabel := "test"
	fullAwsPeeringResourceLabel := fmt.Sprintf("confluent_peering.%s", awsPeeringResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAwsPeeringDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsPeeringConfig(mockServerUrl, awsPeeringDisplayName, awsPeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullAwsPeeringResourceLabel),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "display_name", awsPeeringDisplayName),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
			{
				Config: testAccCheckAwsPeeringConfigWithoutDisplayNameSet(mockServerUrl, awsPeeringResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullAwsPeeringResourceLabel),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "display_name", awsPeeringDisplayName),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPeeringResourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
			{
				ResourceName:      fullAwsPeeringResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					awsPeeringId := resources[fullAwsPeeringResourceLabel].Primary.ID
					environmentId := resources[fullAwsPeeringResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + awsPeeringId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAwsPeeringStub, fmt.Sprintf("POST %s", awsPeeringUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsPeeringStub, fmt.Sprintf("DELETE %s?environment=%s", awsPeeringUrlPath, awsPeeringEnvironmentId), expectedCountOne)
}

func testAccCheckAwsPeeringDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each aws peering is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_peering" {
			continue
		}
		deletedPeeringId := rs.Primary.ID
		req := c.netClient.PeeringsNetworkingV1Api.GetNetworkingV1Peering(c.netApiContext(context.Background()), deletedPeeringId).Environment(awsPeeringEnvironmentId)
		deletedPeering, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedPeering.Id != nil {
			// Otherwise return the error
			if *deletedPeering.Id == rs.Primary.ID {
				return fmt.Errorf("aws peering (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAwsPeeringConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
        display_name = "%s"
	    aws {
		  account = "%s"
          vpc = "%s"
          routes = [%q, %q]
          customer_region = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, awsAccountNumber, awsPeeringVpcId, awsPeeringRoutes[0], awsPeeringRoutes[1],
		awsPeeringCustomerRegion, awsPeeringEnvironmentId, awsPeeringNetworkId)
}

func testAccCheckAwsPeeringConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_peering" "%s" {
	    aws {
		  account = "%s"
          vpc = "%s"
          routes = [%q, %q]
          customer_region = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, awsAccountNumber, awsPeeringVpcId, awsPeeringRoutes[0], awsPeeringRoutes[1],
		awsPeeringCustomerRegion, awsPeeringEnvironmentId, awsPeeringNetworkId)
}

func testAccCheckAwsPeeringExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s AWS Peering has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Aws Peering", n)
		}

		return nil
	}
}
