// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	ipFilterResourceScenarioName = "confluent_ip_filter Resource Lifecycle"

	scenarioStateIpFilterHasBeenCreated = "A new IP group has been created"
	scenarioStateIpFilterHasBeenUpdated = "A IP group has been updated"

	ipFilterResourceLabel = "test"
	newIpFilterId         = "ipf-12345"
)

var fullIpFilterResourceLabel = fmt.Sprintf("confluent_ip_filter.%s", ipFilterResourceLabel)

var createIpFilterUrlPath = "/iam/v2/ip-filters"
var newIpFilterUrlPath = fmt.Sprintf("/iam/v2/ip-filters/%s", newIpFilterId)

func TestAccResourceIpFilter(t *testing.T) {
	// Set TF_ACC environment variable to enable acceptance tests
	t.Setenv("TF_ACC", "1")

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

	// ===== Create stubs =====
	createIpFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/create_ip_filter.json")

	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(createIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(newIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenCreated).
		WillReturn(
			string(createIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// ===== Update stubs =====
	updateIpFilterResponse, _ := ioutil.ReadFile("../testdata/ip_filter/update_ip_filter.json")

	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(newIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenCreated).
		WillSetStateTo(scenarioStateIpFilterHasBeenUpdated).
		WillReturn(
			string(updateIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(newIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenUpdated).
		WillReturn(
			string(updateIpFilterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// ===== Delete stubs =====
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(newIpFilterUrlPath)).
		InScenario(ipFilterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpFilterHasBeenUpdated).
		WillReturn(
			"",
			nil,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			// ===== Create test =====
			{
				Config: testAccResourceIpFilterConfig(
					mockServerUrl,
					ipFilterResourceLabel,
					"Management API Rules",
					"management",
					"crn://confluent.cloud/organization=org-123/environment=env-abc",
					[]string{
						"MANAGEMENT",
						"SCHEMA",
						"FLINK",
					},
					[]string{
						"ipg-12345",
					},
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramId, "ipf-12345"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramFilterName, "Management API Rules"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramResourceGroup, "management"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramResourceScope, "crn://confluent.cloud/organization=org-123/environment=env-abc"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, "operation_groups.#", "3"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "operation_groups.*", "MANAGEMENT"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "operation_groups.*", "SCHEMA"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "operation_groups.*", "FLINK"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, "ip_group_ids.#", "1"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "ip_group_ids.*", "ipg-12345"),
				),
			},
			// ===== Import test =====
			{
				ResourceName:      fullIpFilterResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// ===== Create test =====
			{
				Config: testAccResourceIpFilterConfig(
					mockServerUrl,
					ipFilterResourceLabel,
					"Management API Rules Update",
					"multiple",
					"crn://confluent.cloud/organization=org-123/environment=env-abc",
					[]string{
						"MANAGEMENT",
						"FLINK",
					},
					[]string{
						"ipg-12345",
						"ipg-67890",
					},
				),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramId, "ipf-12345"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramFilterName, "Management API Rules Update"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramResourceGroup, "multiple"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, paramResourceScope, "crn://confluent.cloud/organization=org-123/environment=env-abc"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, "operation_groups.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "operation_groups.*", "MANAGEMENT"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "operation_groups.*", "FLINK"),
					resource.TestCheckResourceAttr(fullIpFilterResourceLabel, "ip_group_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "ip_group_ids.*", "ipg-12345"),
					resource.TestCheckTypeSetElemAttr(fullIpFilterResourceLabel, "ip_group_ids.*", "ipg-67890"),
				),
			},
		},
	})
}

func testAccResourceIpFilterConfig(mockServerUrl, resourceLabel, filterName, resourceGroup, resourceScope string, operationGroups, ipGroupIds []string) string {
	for i, v := range operationGroups {
		operationGroups[i] = fmt.Sprintf("%q", v)
	}

	for i, v := range ipGroupIds {
		ipGroupIds[i] = fmt.Sprintf("%q", v)
	}

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
		ip_group_ids = [
			%s
		]
	}
	`, mockServerUrl, resourceLabel, filterName, resourceGroup, resourceScope, strings.Join(operationGroups, ",\n"), strings.Join(ipGroupIds, ",\n"))
}
