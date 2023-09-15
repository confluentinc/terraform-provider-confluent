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
	scenarioStateRoleBindingHasBeenCreated = "The new role binding has been just created"
	scenarioStateRoleBindingHasBeenDeleted = "The requested role binding has been deleted"
	rolebindingScenarioName                = "confluent_role_binding Resource Lifecycle"
	roleBindingId                          = "rb-OOXL7"
	roleBindingUrlPath                     = "/iam/v2/role-bindings/rb-OOXL7"

	rbPrincipal     = "User:u-vr99n5"
	rbRolename      = "CloudClusterAdmin"
	rbCrn           = "crn://confluent.cloud/organization=0d9c5d94-e4fe-44ec-9cf1-bd99761fca75/environment=env-ym2y0k/cloud-cluster=lkc-xrk0ng"
	rbResourceLabel = "test_rb_resource_label"
)

func TestAccRoleBinding(t *testing.T) {
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
	createRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/create_role_binding.json")
	createRolebindingStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/role-bindings")).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateRoleBindingHasBeenCreated).
		WillReturn(
			string(createRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createRolebindingStub)

	readCreatedRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/read_created_role_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenCreated).
		WillReturn(
			string(readCreatedRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/read_deleted_role_binding.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenDeleted).
		WillReturn(
			string(readDeletedRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	deleteRolebindingStub := wiremock.Delete(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenCreated).
		WillSetStateTo(scenarioStateRoleBindingHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteRolebindingStub)

	fullRbResourceLabel := fmt.Sprintf("confluent_role_binding.%s", rbResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRoleBindingDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRoleBindingConfig(mockServerUrl, rbResourceLabel, rbPrincipal, rbRolename, rbCrn),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleBindingExists(fullRbResourceLabel),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "id", roleBindingId),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "principal", rbPrincipal),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "role_name", rbRolename),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "crn_pattern", rbCrn),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullRbResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createRolebindingStub, "POST /iam/v2/role-bindings", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteRolebindingStub, fmt.Sprintf("DELETE /iam/v2/role-bindings/%s", roleBindingId), expectedCountOne)
}

func testAccCheckRoleBindingDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each role binding is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_role_binding" {
			continue
		}
		deletedRoleBindingId := rs.Primary.ID
		req := c.mdsClient.RoleBindingsIamV2Api.GetIamV2RoleBinding(c.mdsApiContext(context.Background()), deletedRoleBindingId)
		deletedRoleBinding, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedRoleBinding.Id != nil {
			// Otherwise return the error
			if *deletedRoleBinding.Id == rs.Primary.ID {
				return fmt.Errorf("role binding (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckRoleBindingConfig(mockServerUrl, label, principal, roleName, crn string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_role_binding" "%s" {
		principal = "%s"
		role_name = "%s"
		crn_pattern = "%s"
	}
	`, mockServerUrl, label, principal, roleName, crn)
}

func testAccCheckRoleBindingExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s role binding has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s role binding", n)
		}

		return nil
	}
}
