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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	privateLinkAttachmentConnectionAwsResourceScenarioName        = "confluent_private_link_attachment_connection Resource Lifecycle"
	scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenCreated = "A new private link attachment connection has been just created"
	scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenUpdated = "A new private link attachment connection has been just updated"
	scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenDeleted = "A new private link attachment connection has been just deleted"
	scenarioStatePrivateLinkAttachmentConnectionProvisioning      = "A new private link attachment connection is being provisioning"

	privateLinkAttachmentConnectionAwsUrlPath       = "/networking/v1/private-link-attachment-connections"
	privateLinkAttachmentConnectionAwsResourceLabel = "confluent_private_link_attachment_connection.main"
)

func TestAccPrivateLinkAttachmentConnectionAws(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	createPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/create_aws_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionProvisioning).
		WillReturn(
			string(createPlattResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readProvisioningPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_provisioning_aws_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionProvisioning).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenCreated).
		WillReturn(
			string(readProvisioningPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_aws_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenCreated).
		WillReturn(
			string(readPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_updated_aws_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenCreated).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenUpdated).
		WillReturn(
			string(updatedPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenUpdated).
		WillReturn(
			string(updatedPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	/*_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(networkLinkEndpointReadUrlPath)).
	InScenario(privateLinkAttachmentConnectionAwsResourceScenarioName).
	WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionAwsHasBeenDeleted).
	WillReturn(
		"",
		contentTypeJSONHeader,
		http.StatusNotFound,
	))*/

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourcePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "id", "plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-plyvyl/private-link-attachment-connection=plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "display_name", "my_vpc"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "aws.0.vpc_endpoint_id", "vpce-0ed4d51f5d6ef9b6d"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "private_link_attachment.0.id", "platt-plyvyl"),
				),
			},
			{
				Config: testAccCheckResourceUpdatePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "id", "plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-plyvyl/private-link-attachment-connection=plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "display_name", "my_vpc_update"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "aws.0.vpc_endpoint_id", "vpce-0ed4d51f5d6ef9b6d"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsResourceLabel, "private_link_attachment.0.id", "platt-plyvyl"),
				),
			},
		},
	})

	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})
}

func testAccCheckResourcePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource confluent_private_link_attachment_connection main {
	    display_name = "my_vpc"
		environment {
			id = "env-8gv0v5"
		}
		aws {
			vpc_endpoint_id = "vpce-0ed4d51f5d6ef9b6d"
		}
		private_link_attachment {
			id = "platt-plyvyl"
		}
	}
	`, mockServerUrl)
}

func testAccCheckResourceUpdatePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource confluent_private_link_attachment_connection main {
	    display_name = "my_vpc_update"
		environment {
			id = "env-8gv0v5"
		}
		aws {
			vpc_endpoint_id = "vpce-0ed4d51f5d6ef9b6d"
		}
		private_link_attachment {
			id = "platt-plyvyl"
		}
	}
	`, mockServerUrl)
}
