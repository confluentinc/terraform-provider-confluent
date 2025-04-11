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
	scenarioStateGroupMappingHasBeenCreated             = "The new group mapping has been just created"
	scenarioStateGroupMappingDescriptionHaveBeenUpdated = "The new group mapping's description have been just updated"
	scenarioStateGroupMappingHasBeenDeleted             = "The new group mapping has been deleted"
	groupMappingScenarioName                            = "confluent_group_mapping Resource Lifecycle"

	groupMappingId            = "group-w4vP"
	groupMappingResourceLabel = "test_group_mapping_resource_label"
	groupMappingDisplayName   = "Default"
	groupMappingFilter        = "\"engineering\" in groups"
	groupMappingDescription   = "Permission for all users in everyone group"
)

func TestAccGroupMapping(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createGroupMappingResponse, _ := ioutil.ReadFile("../testdata/group_mapping/create_group_mapping.json")
	createGroupMappingStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGroupMappingHasBeenCreated).
		WillReturn(
			string(createGroupMappingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createGroupMappingStub)

	readCreatedGroupMappingResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_created_group_mapping.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(scenarioStateGroupMappingHasBeenCreated).
		WillReturn(
			string(readCreatedGroupMappingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGroupMappingResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_updated_group_mapping.json")
	patchGroupMappingStub := wiremock.Patch(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(scenarioStateGroupMappingHasBeenCreated).
		WillSetStateTo(scenarioStateGroupMappingDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedGroupMappingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchGroupMappingStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(scenarioStateGroupMappingDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedGroupMappingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedGroupMappingResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_deleted_group_mapping.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(scenarioStateGroupMappingHasBeenDeleted).
		WillReturn(
			string(readDeletedGroupMappingResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteGroupMappingStub := wiremock.Delete(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingScenarioName).
		WhenScenarioStateIs(scenarioStateGroupMappingDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateGroupMappingHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteGroupMappingStub)

	// in order to test tf update (step #3)
	groupMappingUpdatedFilter := "\"updated\" in groups"
	groupMappingUpdatedDisplayName := "Default updated"
	groupMappingUpdatedDescription := "Permission for all users in everyone group updated"
	fullGroupMappingResourceLabel := fmt.Sprintf("confluent_group_mapping.%s", groupMappingResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckGroupMappingDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGroupMappingConfig(mockServerUrl, groupMappingResourceLabel, groupMappingDisplayName, groupMappingFilter, groupMappingDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingExists(fullGroupMappingResourceLabel),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "id", groupMappingId),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "display_name", groupMappingDisplayName),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "filter", groupMappingFilter),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "description", groupMappingDescription),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullGroupMappingResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckGroupMappingConfig(mockServerUrl, groupMappingResourceLabel, groupMappingUpdatedDisplayName, groupMappingUpdatedFilter, groupMappingUpdatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingExists(fullGroupMappingResourceLabel),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "id", groupMappingId),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "display_name", groupMappingUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "filter", groupMappingUpdatedFilter),
					resource.TestCheckResourceAttr(fullGroupMappingResourceLabel, "description", groupMappingUpdatedDescription),
				),
			},
			{
				ResourceName:      fullGroupMappingResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createGroupMappingStub, "POST /iam/v2/sso/group-mappings", expectedCountOne)
	checkStubCount(t, wiremockClient, patchGroupMappingStub, "PATCH /iam/v2/sso/group-mappings/group-w4vP", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteGroupMappingStub, "DELETE /iam/v2/sso/group-mappings/group-w4vP", expectedCountOne)

	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckGroupMappingDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each group mapping is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_group_mapping" {
			continue
		}
		deletedGroupMappingId := rs.Primary.ID
		req := c.ssoClient.GroupMappingsIamV2SsoApi.GetIamV2SsoGroupMapping(c.iamApiContext(context.Background()), deletedGroupMappingId)
		deletedGroupMapping, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// iam/v2/sso/{nonExistentGroupMappingId/deletedGroupMappingID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the group mapping is destroyed.
			return nil
		} else if err == nil && deletedGroupMapping.Id != nil {
			// Otherwise return the error
			if *deletedGroupMapping.Id == rs.Primary.ID {
				return fmt.Errorf("group mapping (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckGroupMappingConfig(mockServerUrl, gmResourceLabel, gmDisplayName, gmFilter, gmDescription string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_group_mapping" "%s" {
		display_name = "%s"
		filter       = %q
		description  = "%s"
	}
	`, mockServerUrl, gmResourceLabel, gmDisplayName, gmFilter, gmDescription)
}

func testAccCheckGroupMappingExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s group mapping has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s group mapping", n)
		}

		return nil
	}
}
