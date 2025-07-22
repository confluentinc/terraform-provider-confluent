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
	scenarioStateIpFilterHasBeenCreated = "The new IP Group has been just created"
	scenarioStateIpFilterHasBeenUpdated = "The new IP Group's description have been just updated"
	scenarioStateIpFilterHasBeenDeleted = "The new IP Group has been deleted"
	ipFilterResourceScenarioName        = "confluent_ip_filter Resource Lifecycle"

	testIPFilterID            = "ipf-w7rg0"
	testIpFilterName          = "New Filter"
	testIpFilterResourceGroup = "multiple"
	testIpFilterResourceScope = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa"

	testIpFilterUpdatedName          = "Updated Filter"
	testIpFilterUpdatedResourceGroup = "multiple"
	testIpFilterUpdatedResourceScope = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa"

	testIpFilterResourceLabel = "test_ip_filter_resource_label"
)

var testIpFilterOperationGroups = []string{"MANAGEMENT", "FLINK"}
var testIpFilterIpGroups = []string{"ipg-3o91o"}

var testIpFilterUpdatedOperationGroups = []string{"MANAGEMENT"}
var testIpFilterUpdatedIpGroups = []string{"ipg-3q56d"}

func TestAccIPFilter(t *testing.T) {
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
	createIPFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/create_ip_filter.json")
	createIPFilterStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/ip-filters")).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(createIPFilterResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createIPFilterStub)

	readCreatedIPFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/read_created_ip_filter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(readCreatedIPFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedIPFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/read_updated_ip_filter.json")
	patchSaStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenCreated).
		WillSetStateTo(scenarioStateIpFilterHasBeenUpdated).
		WillReturn(
			string(readUpdatedIPFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchSaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenUpdated).
		WillReturn(
			string(readUpdatedIPFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedIPFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/read_deleted_ip_filter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenDeleted).
		WillReturn(
			string(readDeletedIPFilterResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteIPFilterStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenUpdated).
		WillSetStateTo(scenarioStateIpFilterHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteIPFilterStub)

	// in order to test tf update (step #3)
	fullIPFilterResourceLabel := fmt.Sprintf("confluent_ip_filter.%s", testIpFilterResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIPFilterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIPFilterConfig(mockServerUrl, testIpFilterResourceLabel, testIpFilterName, testIpFilterResourceGroup,
					testIpFilterResourceScope, testIpFilterOperationGroups, testIpFilterIpGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPFilterExists(fullIPFilterResourceLabel),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramId, testIPFilterID),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramFilterName, testIpFilterName),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramResourceGroup, testIpFilterResourceGroup),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramResourceScope, testIpFilterResourceScope),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.#", paramOperationGroups), strconv.Itoa(len(testIpFilterOperationGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.*", paramOperationGroups), testIpFilterOperationGroups[0]),
					resource.TestCheckTypeSetElemAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.*", paramOperationGroups), testIpFilterOperationGroups[1]),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.#", paramIPGroups), strconv.Itoa(len(testIpFilterIpGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.*", paramIPGroups), testIpFilterIpGroups[0]),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullIPFilterResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckIPFilterConfig(mockServerUrl, testIpFilterResourceLabel, testIpFilterUpdatedName, testIpFilterUpdatedResourceGroup,
					testIpFilterUpdatedResourceScope, testIpFilterUpdatedOperationGroups, testIpFilterUpdatedIpGroups),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPFilterExists(fullIPFilterResourceLabel),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramId, testIPFilterID),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramFilterName, testIpFilterUpdatedName),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramResourceGroup, testIpFilterUpdatedResourceGroup),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, paramResourceScope, testIpFilterUpdatedResourceScope),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.#", paramOperationGroups), strconv.Itoa(len(testIpFilterUpdatedOperationGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.*", paramOperationGroups), testIpFilterUpdatedOperationGroups[0]),
					resource.TestCheckResourceAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.#", paramIPGroups), strconv.Itoa(len(testIpFilterUpdatedIpGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPFilterResourceLabel, fmt.Sprintf("%s.*", paramIPGroups), testIpFilterUpdatedIpGroups[0]),
				),
			},
			{
				ResourceName:      fullIPFilterResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createIPFilterStub, "POST /iam/v2/ip-filters", expectedCountOne)
	checkStubCount(t, wiremockClient, patchSaStub, "PATCH /iam/v2/ip-filters/ipg-wyorq", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteIPFilterStub, "DELETE /iam/v2/ip-filters/ipg-wyorq", expectedCountOne)
}

func testAccCheckIPFilterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each IP Group is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_ip_filter" {
			continue
		}
		deletedIPFilterId := rs.Primary.ID
		req := c.iamIPClient.IPFiltersIamV2Api.GetIamV2IpFilter(c.iamIPApiContext(context.Background()), deletedIPFilterId)
		deletedServiceAccount, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/ip-filters/{nonExistentSaId/deletedSaID} returns http.StatusForbidden instead of http.StatusNotFound
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

func testAccCheckIPFilterConfig(mockServerUrl, resourceLabel, filterName, resourceGroup, resourceScope string, operationGroups, ipGroups []string) string {
	quotedOperationGroups := make([]string, len(operationGroups))
	for i, block := range operationGroups {
		quotedOperationGroups[i] = fmt.Sprintf("%q", block)
	}

	joinedQuotedOperationGroups := strings.Join(quotedOperationGroups, ",\n    ")

	quotedIPGroups := make([]string, len(ipGroups))
	for i, block := range ipGroups {
		quotedIPGroups[i] = fmt.Sprintf("%q", block)
	}

	joinedQuotedIPGroups := strings.Join(quotedIPGroups, ",\n    ")

	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_ip_filter" "%s" {
		filter_name = "%s"
		resource_group = "%s"
		resource_scope = "%s"
		operation_groups = [
			%s
		]
		ip_groups = [
			%s
		]
	}
	`, mockServerUrl, resourceLabel, filterName, resourceGroup, resourceScope, joinedQuotedOperationGroups, joinedQuotedIPGroups)
}

func testAccCheckIPFilterExists(n string) resource.TestCheckFunc {
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
