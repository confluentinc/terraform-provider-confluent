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
	"time"
)

const (
	privateLinkAttachmentAzureDataSourceScenarioName = "confluent_private_link_attachment Data Source Lifecycle"

	privateLinkAttachmentAzureReadUrlPath     = "/networking/v1/private-link-attachments/platt-abcdef"
	privateLinkAttachmentAzureId              = "platt-abcdef"
	privateLinkAttachmentAzureDataSourceLabel = "data.confluent_private_link_attachment.main"
)

func TestAccDataSourcePrivateLinkAttachmentAzure(t *testing.T) {
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

	readPrivateLinkAttachmentAzureResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/read_azure_platt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentAzureReadUrlPath)).
		InScenario(privateLinkAttachmentAzureDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPrivateLinkAttachmentAzureResponse),
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
				Config: testAccCheckDataSourcePrivateLinkAttachmentAzureWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "id", "platt-abcdef"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment=platt-abcdef"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "display_name", "prod-aws-us-east1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "region", "us-east-1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "azure.#", "1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "azure.0.private_link_service_alias", "pls-plt-abcdef-az1.f5aedb5a-5830-4ca6-9285-e5c81ffca2cb.centralus.azure.privatelinkservice"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAzureDataSourceLabel, "azure.0.private_link_service_resource_id", "/subscriptions/12345678-9012-3456-7890-123456789012/resourceGroups/rg-abcdef/providers/Microsoft.Network/privateLinkServices/pls-plt-abcdef-az1"),
				),
			},
		},
	})
}

func testAccCheckDataSourcePrivateLinkAttachmentAzureWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
  		endpoint = "%s"
 	}

 	data "confluent_private_link_attachment" "main" {
 		id = "%s"
         environment {
 			id = "env-12345"
 	  	}
 	}
 	`, mockServerUrl, privateLinkAttachmentAzureId)
}
