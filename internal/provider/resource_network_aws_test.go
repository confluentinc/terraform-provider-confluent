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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateAwsNetworkIsProvisioning = "The new aws network is in provisioning state"
	scenarioStateAwsNetworkHasBeenCreated = "The new aws network has been just created"
	scenarioStateAwsNetworkHasBeenDeleted = "The new aws network has been deleted"
	awsNetworkScenarioName                = "confluent_network aws Resource Lifecycle"
	awsNetworkCloud                       = "AWS"
	awsNetworkRegion                      = "us-east-2"
	awsNetworkConnectionType              = "PRIVATELINK"
	awsNetworkEnvironmentId               = "env-gz903"
	awsNetworkId                          = "n-pr1jy6"
	awsDnsDomain                          = "pr1jy6.us-east-2.aws.confluent.cloud"
	awsNetworkVpc                         = "vpc-03e78ba4db7bb1789"
	awsNetworkAccount                     = "012345678901"
	awsNetworkPrivateLinkEndpointService  = "com.amazonaws.vpce.us-east-2.vpce-svc-0089db43e25590123"
	awsNetworkResourceName                = "crn://confluent.cloud/organization=foo/environment=env-gz903/network=n-pr1jy6"

	firstZoneAwsNetwork           = "use2-az1"
	firstZoneSubdomainAwsNetwork  = "use2-az1.pr1jy6.us-east-2.aws.confluent.cloud"
	secondZoneAwsNetwork          = "use2-az2"
	secondZoneSubdomainAwsNetwork = "use2-az2.pr1jy6.us-east-2.aws.confluent.cloud"
	thirdZoneAwsNetwork           = "use2-az3"
	thirdZoneSubdomainAwsNetwork  = "use2-az3.pr1jy6.us-east-2.aws.confluent.cloud"
)

var awsNetworkZones = []string{"use2-az1", "use2-az2", "use2-az3"}

var awsNetworkUrlPath = fmt.Sprintf("/networking/v1/networks/%s", awsNetworkId)

func TestAccAwsNetwork(t *testing.T) {
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
	createAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws/create_network.json")
	createAwsNetworkStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/networks")).
		InScenario(awsNetworkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAwsNetworkIsProvisioning).
		WillReturn(
			string(createAwsNetworkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAwsNetworkStub)

	readProvisioningAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws/read_provisioning_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsNetworkUrlPath)).
		InScenario(awsNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsNetworkIsProvisioning).
		WillSetStateTo(scenarioStateAwsNetworkHasBeenCreated).
		WillReturn(
			string(readProvisioningAwsNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws/read_created_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsNetworkUrlPath)).
		InScenario(awsNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsNetworkHasBeenCreated).
		WillReturn(
			string(readCreatedAwsNetworkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAwsNetworkStub := wiremock.Delete(wiremock.URLPathEqualTo(awsNetworkUrlPath)).
		InScenario(awsNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsNetworkHasBeenCreated).
		WillSetStateTo(scenarioStateAwsNetworkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAwsNetworkStub)

	readDeletedAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws/read_deleted_network.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsNetworkUrlPath)).
		InScenario(awsNetworkScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsNetworkEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsNetworkHasBeenDeleted).
		WillReturn(
			string(readDeletedAwsNetworkResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	awsNetworkDisplayName := "s-n9553"
	awsNetworkResourceLabel := "test"
	fullAwsNetworkResourceLabel := fmt.Sprintf("confluent_network.%s", awsNetworkResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAwsNetworkDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsNetworkConfig(mockServerUrl, awsNetworkDisplayName, awsNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsNetworkExists(fullAwsNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramId, awsNetworkId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDisplayName, awsNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCloud, awsNetworkCloud),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), awsNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), awsNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramRegion, awsNetworkRegion),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), strconv.Itoa(len(awsNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), awsNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), awsNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), awsNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramResourceName, awsNetworkResourceName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDnsDomain, awsDnsDomain),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAwsNetwork), firstZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAwsNetwork), secondZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAwsNetwork), thirdZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramVpc), awsNetworkVpc),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramAccount), awsNetworkAccount),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramPrivateLinkEndpointService), awsNetworkPrivateLinkEndpointService),
				),
			},
			{
				Config: testAccCheckAwsNetworkConfigWithoutDisplayNameAndZonesSet(mockServerUrl, awsNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsNetworkExists(fullAwsNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramId, awsNetworkId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDisplayName, awsNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCloud, awsNetworkCloud),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), awsNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), awsNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramRegion, awsNetworkRegion),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), strconv.Itoa(len(awsNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), awsNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), awsNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), awsNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramResourceName, awsNetworkResourceName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDnsDomain, awsDnsDomain),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, firstZoneAwsNetwork), firstZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, secondZoneAwsNetwork), secondZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, thirdZoneAwsNetwork), thirdZoneSubdomainAwsNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramVpc), awsNetworkVpc),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramAccount), awsNetworkAccount),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramPrivateLinkEndpointService), awsNetworkPrivateLinkEndpointService),
				),
			},
			{
				ResourceName:      fullAwsNetworkResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					awsNetworkId := resources[fullAwsNetworkResourceLabel].Primary.ID
					environmentId := resources[fullAwsNetworkResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + awsNetworkId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAwsNetworkStub, fmt.Sprintf("POST %s", azureNetworkUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsNetworkStub, fmt.Sprintf("DELETE %s?environment=%s", azureNetworkUrlPath, awsNetworkEnvironmentId), expectedCountOne)
}

func testAccCheckAwsNetworkDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each aws network is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_network" {
			continue
		}
		deletedAwsNetworkId := rs.Primary.ID
		req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(context.Background()), deletedAwsNetworkId).Environment(awsNetworkEnvironmentId)
		deletedAwsNetwork, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedAwsNetwork.Id != nil {
			// Otherwise return the error
			if *deletedAwsNetwork.Id == rs.Primary.ID {
				return fmt.Errorf("aws network (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAwsNetworkConfig(mockServerUrl, networkDisplayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_network" "%s" {
        display_name     = "%s"
	    cloud            = "%s"
	    region           = "%s"
        zones            = [%q, %q, %q]
	    connection_types = ["%s"]
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, networkDisplayName, awsNetworkCloud, awsNetworkRegion, awsNetworkZones[0],
		awsNetworkZones[1], awsNetworkZones[2], awsNetworkConnectionType, awsNetworkEnvironmentId)
}

func testAccCheckAwsNetworkConfigWithoutDisplayNameAndZonesSet(mockServerUrl, resourceLabel string) string {
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
	`, mockServerUrl, resourceLabel, awsNetworkCloud, awsNetworkRegion, awsNetworkConnectionType, awsNetworkEnvironmentId)
}

func testAccCheckAwsNetworkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s aws network has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s aws network", n)
		}

		return nil
	}
}
