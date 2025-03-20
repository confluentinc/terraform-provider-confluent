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
	privateLinkAttachmentConnectionGcpDataSourceScenarioName = "confluent_private_link_attachment_connection Data Source Lifecycle"

	privateLinkAttachmentConnectionGcpReadUrlPath     = "/networking/v1/private-link-attachment-connections/plattc-xyzuvw1"
	privateLinkAttachmentConnectionGcpId              = "plattc-xyzuvw1"
	privateLinkAttachmentConnectionGcpDataSourceLabel = "data.confluent_private_link_attachment_connection.main"
)

func TestAccDataSourcePrivateLinkAttachmentConnectionGcp(t *testing.T) {
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

	readPrivateLinkAttachmentConnectionGcpResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_gcp_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionGcpReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionGcpDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPrivateLinkAttachmentConnectionGcpResponse),
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
				Config: testAccCheckDataSourcePrivateLinkAttachmentConnectionGcpWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "id", "plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment-connection=plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "display_name", "prod-gcp-us-central1-connection"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "gcp.0.private_service_connect_connection_id", "1234567891234"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpDataSourceLabel, "private_link_attachment.0.id", "platt-abcdef"),
				),
			},
		},
	})
}

func testAccCheckDataSourcePrivateLinkAttachmentConnectionGcpWithIdSet(mockServerUrl string) string {
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
 	`, mockServerUrl, privateLinkAttachmentConnectionGcpId)
}
