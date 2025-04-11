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
	privateLinkAttachmentConnectionAzureDataSourceScenarioName = "confluent_private_link_attachment_connection Data Source Lifecycle"

	privateLinkAttachmentConnectionAzureReadUrlPath     = "/networking/v1/private-link-attachment-connections/plattc-xyzuvw1"
	privateLinkAttachmentConnectionAzureId              = "plattc-xyzuvw1"
	privateLinkAttachmentConnectionAzureDataSourceLabel = "data.confluent_private_link_attachment_connection.main"
)

func TestAccDataSourcePrivateLinkAttachmentConnectionAzure(t *testing.T) {
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

	readPrivateLinkAttachmentConnectionAzureResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_azure_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAzureReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAzureDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPrivateLinkAttachmentConnectionAzureResponse),
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
				Config: testAccCheckDataSourcePrivateLinkAttachmentConnectionAzureWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "id", "plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment-connection=plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "display_name", "prod-azure-central-us-az1-connection"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "azure.0.private_endpoint_resource_id", "/subscriptions/123aaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-platt-abcdef-az1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAzureDataSourceLabel, "private_link_attachment.0.id", "platt-abcdef"),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourcePrivateLinkAttachmentConnectionAzureWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
  		endpoint = "%s"
 	}

 	data "confluent_private_link_attachment_connection" "main" {
 		id = "%s"
         environment {
 			id = "env-12345"
 	  	}
 	}
 	`, mockServerUrl, privateLinkAttachmentConnectionAzureId)
}
