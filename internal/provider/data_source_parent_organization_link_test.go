// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	parentOrganizationLinkDataSourceScenarioName = "confluent_parent_organization_link Data Source Lifecycle"
	parentOrganizationLinkDataSourceLabel        = "test_parent_organization_link_data_source_label"

	expectedParentOrganizationLinkId       = "78acf398-ef5d-4373-ac40-e2f99ca8f991"
	expectedParentOrganizationLinkParentId = "98c3e8af-c392-418a-b87e-af33c7853b1f"
	expectedParentOrganizationLinkOrgId    = "085d3340-aef4-41e7-b537-9804d34e18fc"
)

var fullParentOrganizationLinkDataSourceLabel = fmt.Sprintf("data.confluent_parent_organization_link.%s", parentOrganizationLinkDataSourceLabel)

func TestAccDataSourceParentOrganizationLink(t *testing.T) {
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

	readParentOrganizationLinkResponse, _ := ioutil.ReadFile("../testdata/parent_organization_link/read_created_link.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/parent-organization-links/78acf398-ef5d-4373-ac40-e2f99ca8f991")).
		InScenario(parentOrganizationLinkDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readParentOrganizationLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceParentOrganizationLinkConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckParentOrganizationLinkExists(fullParentOrganizationLinkDataSourceLabel),
					resource.TestCheckResourceAttr(fullParentOrganizationLinkDataSourceLabel, "id", expectedParentOrganizationLinkId),
					resource.TestCheckResourceAttr(fullParentOrganizationLinkDataSourceLabel, "parent.#", "1"),
					resource.TestCheckResourceAttr(fullParentOrganizationLinkDataSourceLabel, "parent.0.id", expectedParentOrganizationLinkParentId),
					resource.TestCheckResourceAttr(fullParentOrganizationLinkDataSourceLabel, "organization.#", "1"),
					resource.TestCheckResourceAttr(fullParentOrganizationLinkDataSourceLabel, "organization.0.id", expectedParentOrganizationLinkOrgId),
				),
			},
		},
	})
}

func testAccCheckDataSourceParentOrganizationLinkConfig(confluentCloudBaseUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
      endpoint = "%s"
    }
    data "confluent_parent_organization_link" "%s" {
        id = "78acf398-ef5d-4373-ac40-e2f99ca8f991"
    }
    `, confluentCloudBaseUrl, parentOrganizationLinkDataSourceLabel)
}
