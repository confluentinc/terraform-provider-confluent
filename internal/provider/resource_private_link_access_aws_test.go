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
	scenarioStateAwsPlaIsProvisioning          = "The new aws private link access is provisioning"
	scenarioStateAwsPlaIsDeprovisioning        = "The new aws private link access is deprovisioning"
	scenarioStateAwsPlaHasBeenCreated          = "The new aws private link access has been just created"
	scenarioStateAwsPlaIsInDeprovisioningState = "The new aws private link access is in deprovisioning state"
	scenarioStateAwsPlaHasBeenDeleted          = "The new aws private link access's deletion has been just completed"
	awsPlaScenarioName                         = "confluent_private_link_access Resource Lifecycle"
	awsPlaEnvironmentId                        = "env-5wyjmz"
	awsPlaNetworkId                            = "n-5p59z6"
	awsPlaId                                   = "pla-3prjy6"
	awsAccountNumber                           = "012345678901"
)

var awsPlaUrlPath = fmt.Sprintf("/networking/v1/private-link-accesses/%s", awsPlaId)

func TestAccAwsPrivateLinkAccess(t *testing.T) {
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
	createAwsPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/create_pla.json")
	createAwsPlaStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/private-link-accesses")).
		InScenario(awsPlaScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAwsPlaIsProvisioning).
		WillReturn(
			string(createAwsPlaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createAwsPlaStub)

	readProvisioningAwsPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_provisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(awsPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPlaIsProvisioning).
		WillSetStateTo(scenarioStateAwsPlaHasBeenCreated).
		WillReturn(
			string(readProvisioningAwsPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAwsPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_created_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(awsPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPlaHasBeenCreated).
		WillReturn(
			string(readCreatedAwsPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteAwsPlaStub := wiremock.Delete(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(awsPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPlaHasBeenCreated).
		WillSetStateTo(scenarioStateAwsPlaIsInDeprovisioningState).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteAwsPlaStub)

	readDeprovisioningAwsPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_deprovisioning_pla.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(awsPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPlaIsDeprovisioning).
		WillSetStateTo(scenarioStateAwsPlaHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAwsPlaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAwsPlaResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_deleted_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(awsPlaScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsPlaHasBeenDeleted).
		WillReturn(
			string(readDeletedAwsPlaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	awsPlaDisplayName := "prod-pl-use2"
	awsPlaResourceLabel := "test"
	fullAwsPlaResourceLabel := fmt.Sprintf("confluent_private_link_access.%s", awsPlaResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAwsPlaDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsPlaConfig(mockServerUrl, awsPlaDisplayName, awsPlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPlaExists(fullAwsPlaResourceLabel),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "id", awsPlaId),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "display_name", awsPlaDisplayName),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "environment.0.id", awsPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "network.0.id", awsPlaNetworkId),
				),
			},
			{
				Config: testAccCheckAwsPlaConfigWithoutDisplayNameSet(mockServerUrl, awsPlaResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPlaExists(fullAwsPlaResourceLabel),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "id", awsPlaId),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "display_name", awsPlaDisplayName),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "environment.0.id", awsPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAwsPlaResourceLabel, "network.0.id", awsPlaNetworkId),
				),
			},
			{
				ResourceName:      fullAwsPlaResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					awsPlaId := resources[fullAwsPlaResourceLabel].Primary.ID
					environmentId := resources[fullAwsPlaResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + awsPlaId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAwsPlaStub, fmt.Sprintf("POST %s", awsPlaUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsPlaStub, fmt.Sprintf("DELETE %s?environment=%s", awsPlaUrlPath, awsPlaEnvironmentId), expectedCountOne)
}

func testAccCheckAwsPlaDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each private link access is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_private_link_access" {
			continue
		}
		deletedPrivateLinkAccessId := rs.Primary.ID
		req := c.netClient.PrivateLinkAccessesNetworkingV1Api.GetNetworkingV1PrivateLinkAccess(c.netApiContext(context.Background()), deletedPrivateLinkAccessId).Environment(awsPlaEnvironmentId)
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

func testAccCheckAwsPlaConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
        display_name = "%s"
	    aws {
		  account = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, awsAccountNumber, awsPlaEnvironmentId, awsPlaNetworkId)
}

func testAccCheckAwsPlaConfigWithoutDisplayNameSet(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_private_link_access" "%s" {
	    aws {
		  account = "%s"
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, awsAccountNumber, awsPlaEnvironmentId, awsPlaNetworkId)
}

func testAccCheckAwsPlaExists(n string) resource.TestCheckFunc {
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
