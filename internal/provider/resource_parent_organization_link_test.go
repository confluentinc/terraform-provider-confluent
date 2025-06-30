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
	scenarioStateParentOrgLinkHasBeenCreated = "The new parent organization link has been just created"
	scenarioStateParentOrgLinkHasBeenDeleted = "The requested parent organization link has been deleted"
	parentOrgLinkScenarioName                = "confluent_parent_organization_link Resource Lifecycle"
	parentOrgLinkUrlPath                     = "/iam/v2/parent-organization-links/78acf398-ef5d-4373-ac40-e2f99ca8f991"

	testParentOrgLinkId             = "78acf398-ef5d-4373-ac40-e2f99ca8f991"
	testParentOrgLinkParentId       = "98c3e8af-c392-418a-b87e-af33c7853b1f"
	testParentOrgLinkOrganizationId = "085d3340-aef4-41e7-b537-9804d34e18fc"
	testParentOrgLinkResourceLabel  = "test_parent_org_link_resource_label"
)

func TestAccParentOrganizationLink(t *testing.T) {
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
	createParentOrgLinkResponse, _ := ioutil.ReadFile("../testdata/parent_organization_link/create_link.json")
	createParentOrgLinkStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/parent-organization-links")).
		InScenario(parentOrgLinkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateParentOrgLinkHasBeenCreated).
		WillReturn(
			string(createParentOrgLinkResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createParentOrgLinkStub)

	readCreatedParentOrgLinkResponse, _ := ioutil.ReadFile("../testdata/parent_organization_link/read_created_link.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(parentOrgLinkUrlPath)).
		InScenario(parentOrgLinkScenarioName).
		WhenScenarioStateIs(scenarioStateParentOrgLinkHasBeenCreated).
		WillReturn(
			string(readCreatedParentOrgLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedParentOrgLinkResponse, _ := ioutil.ReadFile("../testdata/parent_organization_link/read_deleted_link.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(parentOrgLinkUrlPath)).
		InScenario(parentOrgLinkScenarioName).
		WhenScenarioStateIs(scenarioStateParentOrgLinkHasBeenDeleted).
		WillReturn(
			string(readDeletedParentOrgLinkResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	deleteParentOrgLinkStub := wiremock.Delete(wiremock.URLPathEqualTo(parentOrgLinkUrlPath)).
		InScenario(parentOrgLinkScenarioName).
		WhenScenarioStateIs(scenarioStateParentOrgLinkHasBeenCreated).
		WillSetStateTo(scenarioStateParentOrgLinkHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteParentOrgLinkStub)

	fullPolResourceLabel := fmt.Sprintf("confluent_parent_organization_link.%s", testParentOrgLinkResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckParentOrganizationLinkDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckParentOrganizationLinkConfig(mockServerUrl, testParentOrgLinkResourceLabel, testParentOrgLinkParentId, testParentOrgLinkOrganizationId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckParentOrganizationLinkExists(fullPolResourceLabel),
					resource.TestCheckResourceAttr(fullPolResourceLabel, "id", testParentOrgLinkId),
					resource.TestCheckResourceAttr(fullPolResourceLabel, "parent.#", "1"),
					resource.TestCheckResourceAttr(fullPolResourceLabel, "parent.0.id", testParentOrgLinkParentId),
					resource.TestCheckResourceAttr(fullPolResourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullPolResourceLabel, "organization.0.id", testParentOrgLinkOrganizationId),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullPolResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createParentOrgLinkStub, "POST /iam/v2/parent-organization-links", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteParentOrgLinkStub, fmt.Sprintf("DELETE /iam/v2/parent-organization-links/%s", testParentOrgLinkId), expectedCountOne)
}

func testAccCheckParentOrganizationLinkDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each parent organization link is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_parent_organization_link" {
			continue
		}
		deletedParentOrgLinkId := rs.Primary.ID
		req := c.parentClient.ParentOrganizationLinksIamV2Api.GetIamV2ParentOrganizationLink(c.parentApiContext(context.Background()), deletedParentOrgLinkId)
		deletedParentOrgLink, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedParentOrgLink.Id != nil {
			// Otherwise return the error
			if *deletedParentOrgLink.Id == rs.Primary.ID {
				return fmt.Errorf("parent organization link (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckParentOrganizationLinkConfig(mockServerUrl, label, parentId, organizationId string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }
    resource "confluent_parent_organization_link" "%s" {
        parent { 
            id = "%s"
        }
        organization { 
            id = "%s"
        }
    }
    `, mockServerUrl, label, parentId, organizationId)
}

func testAccCheckParentOrganizationLinkExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s parent organization link has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s parent organization link", n)
		}

		return nil
	}
}
