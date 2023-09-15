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
	awsPeeringNetworkConnectionType      = connectionTypePeering
	awsPeeringNetworkRegion              = "us-east-1"
	firstZoneSubdomainAwsPeeringNetwork  = "use1-az2.pr1jy6.us-east-1.aws.confluent.cloud"
	secondZoneSubdomainAwsPeeringNetwork = "use1-az5.pr1jy6.us-east-1.aws.confluent.cloud"
	thirdZoneSubdomainAwsPeeringNetwork  = "use1-az6.pr1jy6.us-east-1.aws.confluent.cloud"

	awsPeeringNetworkCidr         = "255.254.0.0/16"
	awsPeeringNetworkReservedCidr = "172.20.255.0/24"
)

var awsPeeringNetworkZones = []string{"use1-az2",
	"use1-az5",
	"use1-az6",
}

var awsPeeringNetworkCidrs = []string{
	"10.2.16.0/27",
	"10.2.16.32/27",
	"10.2.16.64/27",
}

func TestAccAwsZoneInfoNetwork(t *testing.T) {
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
	createAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws_zone_info/create_network.json")
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

	readProvisioningAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws_zone_info/read_provisioning_network.json")
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

	readCreatedAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws_zone_info/read_created_network.json")
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

	readDeletedAwsNetworkResponse, _ := ioutil.ReadFile("../testdata/network/aws_zone_info/read_deleted_network.json")
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
				Config: testAccCheckAwsPeeringNetworkConfig(mockServerUrl, awsNetworkDisplayName, awsNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsNetworkExists(fullAwsNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramId, awsNetworkId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDisplayName, awsNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCloud, awsNetworkCloud),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), awsPeeringNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), awsNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramRegion, awsPeeringNetworkRegion),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), strconv.Itoa(len(awsPeeringNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), awsPeeringNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), awsPeeringNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), awsPeeringNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramDnsConfig), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramDnsConfig, paramResolution), ""),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramResourceName, awsNetworkResourceName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDnsDomain, awsDnsDomain),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCidr, awsPeeringNetworkCidr),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramReservedCidr, awsPeeringNetworkReservedCidr),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZoneInfo), strconv.Itoa(len(awsPeeringNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[0]), firstZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[1]), secondZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[2]), thirdZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramVpc), awsNetworkVpc),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramAccount), awsNetworkAccount),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramPrivateLinkEndpointService), ""),
				),
			},
			{
				Config: testAccCheckAwsPeeringNetworkConfigWithZoneIdSet(mockServerUrl, awsNetworkDisplayName, awsNetworkResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsNetworkExists(fullAwsNetworkResourceLabel),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramId, awsNetworkId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDisplayName, awsNetworkDisplayName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCloud, awsNetworkCloud),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramConnectionTypes), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramConnectionTypes), awsPeeringNetworkConnectionType),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), awsNetworkEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramRegion, awsPeeringNetworkRegion),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZones), strconv.Itoa(len(awsPeeringNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0", paramZones), awsPeeringNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1", paramZones), awsPeeringNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2", paramZones), awsPeeringNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramDnsConfig), "1"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramDnsConfig, paramResolution), ""),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramResourceName, awsNetworkResourceName),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramDnsDomain, awsDnsDomain),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramCidr, awsPeeringNetworkCidr),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, paramReservedCidr, awsPeeringNetworkReservedCidr),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.#", paramZoneInfo), strconv.Itoa(len(awsPeeringNetworkZones))),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[0]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.1.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[1]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2.%s", paramZoneInfo, paramZoneId), awsPeeringNetworkZones[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.2.%s", paramZoneInfo, paramCidr), awsPeeringNetworkCidrs[2]),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, "zonal_subdomains.%", "3"),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[0]), firstZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[1]), secondZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.%s", paramZonalSubdomains, awsPeeringNetworkZones[2]), thirdZoneSubdomainAwsPeeringNetwork),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramVpc), awsNetworkVpc),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramAccount), awsNetworkAccount),
					resource.TestCheckResourceAttr(fullAwsNetworkResourceLabel, fmt.Sprintf("%s.0.%s", paramAws, paramPrivateLinkEndpointService), ""),
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

	checkStubCount(t, wiremockClient, createAwsNetworkStub, fmt.Sprintf("POST %s", awsNetworkUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsNetworkStub, fmt.Sprintf("DELETE %s?environment=%s", awsNetworkUrlPath, awsNetworkEnvironmentId), expectedCountOne)
}

func testAccCheckAwsPeeringNetworkConfig(mockServerUrl, networkDisplayName, resourceLabel string) string {
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
        cidr             = "%s"
        reserved_cidr    = "%s"
	    environment {
		  id = "%s"
	    }
        zone_info {
		  cidr = "%s"
	    }
        zone_info {
		  cidr = "%s"
	    }
        zone_info {
		  cidr = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, networkDisplayName, awsNetworkCloud, awsPeeringNetworkRegion, awsPeeringNetworkZones[0],
		awsPeeringNetworkZones[1], awsPeeringNetworkZones[2], awsPeeringNetworkConnectionType,
		awsPeeringNetworkCidr, awsPeeringNetworkReservedCidr,
		awsNetworkEnvironmentId, awsPeeringNetworkCidrs[0],
		awsPeeringNetworkCidrs[1], awsPeeringNetworkCidrs[2])
}

func testAccCheckAwsPeeringNetworkConfigWithZoneIdSet(mockServerUrl, networkDisplayName, resourceLabel string) string {
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
        cidr             = "%s"
        reserved_cidr    = "%s"
	    environment {
		  id = "%s"
	    }
        zone_info {
		  zone_id = "%s"
		  cidr = "%s"
	    }
        zone_info {
		  zone_id = "%s"
		  cidr = "%s"
	    }
        zone_info {
		  zone_id = "%s"
		  cidr = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, networkDisplayName, awsNetworkCloud, awsPeeringNetworkRegion, awsPeeringNetworkZones[0],
		awsPeeringNetworkZones[1], awsPeeringNetworkZones[2], awsPeeringNetworkConnectionType,
		awsPeeringNetworkCidr, awsPeeringNetworkReservedCidr,
		awsNetworkEnvironmentId, awsPeeringNetworkZones[0], awsPeeringNetworkCidrs[0],
		awsPeeringNetworkZones[1], awsPeeringNetworkCidrs[1], awsPeeringNetworkZones[2], awsPeeringNetworkCidrs[2])
}
