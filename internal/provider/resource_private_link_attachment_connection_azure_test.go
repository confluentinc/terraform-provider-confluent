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
	privateLinkAttachmentConnectionAzureResourceScenarioName        = "confluent_private_link_attachment_connection Resource Lifecycle"
	scenarioStatePrivateLinkAttachmentConnectionAzureHasBeenCreated = "A new private link attachment connection has been just created"

	privateLinkAttachmentConnectionAzureUrlPath       = "/networking/v1/private-link-attachment-connections"
	privateLinkAttachmentConnectionAzureResourceLabel = "confluent_private_link_attachment_connection.main"
)

func TestAccPrivateLinkAttachmentConnectionAzure(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	createPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/create_azure_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAzureUrlPath)).
		InScenario(privateLinkAttachmentConnectionAzureResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionAzureHasBeenCreated).
		WillReturn(
			string(createPlattResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_azure_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAzureReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAzureResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionAzureHasBeenCreated).
		WillReturn(
			string(readPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAzureReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAzureResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourcePrivateLinkAttachmentConnectionAzureWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "id", "plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment-connection=plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "display_name", "prod-azure-central-us-az1-connection"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "azure.0.private_endpoint_resource_id", "/subscriptions/123aaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-platt-abcdef-az1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureResourceLabel, "private_link_attachment.0.id", "platt-abcdef"),
				),
			},
		},
	})
}

func testAccCheckResourcePrivateLinkAttachmentConnectionAzureWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource confluent_private_link_attachment_connection main {
	    display_name = "prod-azure-central-us-az1-connection"
		environment {
			id = "env-12345"
		}
		azure {
			private_endpoint_resource_id = "/subscriptions/123aaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-platt-abcdef-az1"
		}
		private_link_attachment {
			id = "platt-abcdef"
		}
	}
	`, mockServerUrl)
}
