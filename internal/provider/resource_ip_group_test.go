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
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateIpGroupHasBeenCreated = "The new IP Group has been just created"
	scenarioStateIpGroupHasBeenUpdated = "The new IP Group's description have been just updated"
	scenarioStateIpGroupHasBeenDeleted = "The new IP Group has been deleted"
	ipGroupResourceScenarioName        = "confluent_ip_group Resource Lifecycle"

	testIPGroupID            = "ipg-wyorq"
	testIPGroupName          = "CorpNet"
	testIPGroupResourceLabel = "test_ip_group_resource_label"
)

var testIPGroupCidrBlocks = []string{"192.168.0.0/24", "192.168.7.0/24"}

func TestAccIPGroup(t *testing.T) {
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
	createIPGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/create_ip_group.json")
	createIPGroupStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/ip-groups")).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIpGroupHasBeenCreated).
		WillReturn(
			string(createIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createIPGroupStub)

	readCreatedIPGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/read_created_ip_group.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenCreated).
		WillReturn(
			string(readCreatedIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedIPGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/read_updated_ip_group.json")
	patchSaStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenCreated).
		WillSetStateTo(scenarioStateIpGroupHasBeenUpdated).
		WillReturn(
			string(readUpdatedIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchSaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenUpdated).
		WillReturn(
			string(readUpdatedIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedIPGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/read_deleted_ip_group.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenDeleted).
		WillReturn(
			string(readDeletedIPGroupResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteIPGroupStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenUpdated).
		WillSetStateTo(scenarioStateIpGroupHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteIPGroupStub)

	// in order to test tf update (step #3)
	ipGroupUpdatedName := "UpdatedCorpNet"

	testIPGroupUpdatedCidrBlocks := []string{"10.0.1.0/24", "172.16.5.0/24"}
	fullIPGroupResourceLabel := fmt.Sprintf("confluent_ip_group.%s", testIPGroupResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIPGroupDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIPGroupConfig(mockServerUrl, testIPGroupResourceLabel, testIPGroupName, testIPGroupCidrBlocks),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupExists(fullIPGroupResourceLabel),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramId, testIPGroupID),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramGroupName, testIPGroupName),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.#", paramCidrBlocks), strconv.Itoa(len(testIPGroupCidrBlocks))),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramCidrBlocks), testIPGroupCidrBlocks[0]),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramCidrBlocks), testIPGroupCidrBlocks[1]),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullIPGroupResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckIPGroupConfig(mockServerUrl, testIPGroupResourceLabel, ipGroupUpdatedName, testIPGroupUpdatedCidrBlocks),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupExists(fullIPGroupResourceLabel),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramId, testIPGroupID),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramGroupName, ipGroupUpdatedName),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.#", paramCidrBlocks), strconv.Itoa(len(testIPGroupUpdatedCidrBlocks))),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramCidrBlocks), testIPGroupUpdatedCidrBlocks[0]),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramCidrBlocks), testIPGroupUpdatedCidrBlocks[1]),
				),
			},
			{
				ResourceName:      fullIPGroupResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createIPGroupStub, "POST /iam/v2/ip-groups", expectedCountOne)
	checkStubCount(t, wiremockClient, patchSaStub, "PATCH /iam/v2/ip-groups/ipg-wyorq", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteIPGroupStub, "DELETE /iam/v2/ip-groups/ipg-wyorq", expectedCountOne)
}

func testAccCheckIPGroupDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each IP Group is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_ip_group" {
			continue
		}
		deletedIPGroupId := rs.Primary.ID
		req := c.iamIPClient.IPGroupsIamV2Api.GetIamV2IpGroup(c.iamIPApiContext(context.Background()), deletedIPGroupId)
		deletedServiceAccount, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/ip-groups/{nonExistentSaId/deletedSaID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the IP Group is destroyed.
			return nil
		} else if err == nil && deletedServiceAccount.Id != nil {
			// Otherwise return the error
			if *deletedServiceAccount.Id == rs.Primary.ID {
				return fmt.Errorf("IP Group (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckIPGroupConfig(mockServerUrl, resourceLabel, groupName string, cidrBlocks []string) string {
	quotedBlocks := make([]string, len(cidrBlocks))
	for i, block := range cidrBlocks {
		quotedBlocks[i] = fmt.Sprintf("%q", block)
	}

	// Indent for Terraform readability
	joinedBlocks := strings.Join(quotedBlocks, ",\n    ")

	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = [
			%s
		]
	}
	`, mockServerUrl, resourceLabel, groupName, joinedBlocks)
}

func testAccCheckIPGroupExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s IP Group has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s IP Group", n)
		}

		return nil
	}
}
