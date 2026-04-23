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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
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
	if err := wiremockClient.StubFor(createRolebindingStub); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	readCreatedRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/read_created_role_binding.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenCreated).
		WillReturn(
			string(readCreatedRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	readDeletedRolebindingResponse, _ := ioutil.ReadFile("../testdata/role_binding/read_deleted_role_binding.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenDeleted).
		WillReturn(
			string(readDeletedRolebindingResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	deleteRolebindingStub := wiremock.Delete(wiremock.URLPathEqualTo(roleBindingUrlPath)).
		InScenario(rolebindingScenarioName).
		WhenScenarioStateIs(scenarioStateRoleBindingHasBeenCreated).
		WillSetStateTo(scenarioStateRoleBindingHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	if err := wiremockClient.StubFor(deleteRolebindingStub); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	fullRbResourceLabel := fmt.Sprintf("confluent_role_binding.%s", rbResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRoleBindingDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckRoleBindingConfig(mockServerUrl, rbResourceLabel, rbPrincipal, rbRolename, rbCrn, testShouldDisableBefore),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleBindingExists(fullRbResourceLabel),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "id", roleBindingId),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "principal", rbPrincipal),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "role_name", rbRolename),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "crn_pattern", rbCrn),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "disable_wait_for_ready", "false"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullRbResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckRoleBindingConfig(mockServerUrl, rbResourceLabel, rbPrincipal, rbRolename, rbCrn, testShouldDisableAfter),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRoleBindingExists(fullRbResourceLabel),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "id", roleBindingId),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "principal", rbPrincipal),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "role_name", rbRolename),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "crn_pattern", rbCrn),
					resource.TestCheckResourceAttr(fullRbResourceLabel, "disable_wait_for_ready", "true"),
				),
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
		req := c.mdsV2Client.RoleBindingsIamV2Api.GetIamV2RoleBinding(c.mdsV2ApiContext(context.Background()), deletedRoleBindingId)
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

func testAccCheckRoleBindingConfig(mockServerUrl, label, principal, roleName, crn, shouldDisable string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_role_binding" "%s" {
		principal = "%s"
		role_name = "%s"
		crn_pattern = "%s"
		disable_wait_for_ready = %s
	}
	`, mockServerUrl, label, principal, roleName, crn, shouldDisable)
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
