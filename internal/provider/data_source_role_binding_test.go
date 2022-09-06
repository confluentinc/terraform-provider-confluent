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
)

const (
	roleBindingDataSourceScenarioName = "confluent_role_binding Data Source Lifecycle"
)

func TestAccDataSourceRoleBinding(t *testing.T) {
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

	readCreatedRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/read_created_role_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(roleBindingDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullRbDataSourceLabel := fmt.Sprintf("data.confluent_role_binding.%s", rbResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRoleBindingDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleBindingExists(fullRbDataSourceLabel),
					resource.TestCheckResourceAttr(fullRbDataSourceLabel, "id", roleBindingId),
					resource.TestCheckResourceAttr(fullRbDataSourceLabel, "principal", rbPrincipal),
					resource.TestCheckResourceAttr(fullRbDataSourceLabel, "role_name", rbRolename),
					resource.TestCheckResourceAttr(fullRbDataSourceLabel, "crn_pattern", rbCrn),
				),
			},
		},
	})
}

func testAccCheckRoleBindingDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_role_binding" "%s" {
		id = "%s"
	}
	`, mockServerUrl, rbResourceLabel, roleBindingId)
}
