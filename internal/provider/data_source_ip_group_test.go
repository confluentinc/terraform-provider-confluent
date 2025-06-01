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
	ipGroupDataSourceScenarioName = "confluent_ip_group Data Source Lifecycle"

	ipGroupUrlPath = "/iam/v2/ip-groups/ipg-12345"
	ipGroupId      = "ipg-12345"
)

func TestAccDataSourceIpGroup(t *testing.T) {
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

	readIpGroupResponse, _ := ioutil.ReadFile("../testdata/ip_group/read_ip_group.json")

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(ipGroupUrlPath)).
		InScenario(ipGroupDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
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
				Config: testAccDataSourceIpGroupConfig(mockServerUrl, "test", "ipg-12345"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("test", "id", "ipg-12345"),
				),
			},
		},
	})
}

func testAccDataSourceIpGroupConfig(mockServerUrl, resourceLabel, id string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_ip_group" "%s" {
		id = "%s"
	}
	`, mockServerUrl, resourceLabel, id)
}
