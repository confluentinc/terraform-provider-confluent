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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	ipGroupResourceScenarioName = "confluent_ip_group Resource Lifecycle"

	scenarioStateIpGroupHasBeenCreated = "A new IP group has been created"

	createIpGroupUrlPath      = "/iam/v2/ip-groups"
	readCreatedIpGroupUrlPath = "/iam/v2/ip-groups/ipg-12345"
)

func TestAccResourceIpGroup(t *testing.T) {
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

	createIpGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/create_ip_group.json")

	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createIpGroupUrlPath)).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIpGroupHasBeenCreated).
		WillReturn(
			string(createIpGroupResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readIpGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/read_ip_group.json")

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedIpGroupUrlPath)).
		InScenario(ipGroupResourceScenarioName).
		WhenScenarioStateIs(scenarioStateIpGroupHasBeenCreated).
		WillReturn(
			string(readIpGroupResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccResourceIpGroupConfig(mockServerUrl, "test", "Group Name", "192.168.0.0/24"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test", "id", "ipg-12345"),
				),
			},
		},
	})
}

func testAccResourceIpGroupConfig(mockServerUrl, resourceLabel, groupName, cidrBlock string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_ip_group" "%s" {
		group_name = "%s"
		cidr_blocks = ["%s"]
	}
	`, mockServerUrl, resourceLabel, groupName, cidrBlock)
}
