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
	privateLinkAttachmentConnectionGcpResourceScenarioName        = "confluent_private_link_attachment_connection Resource Lifecycle"
	scenarioStatePrivateLinkAttachmentConnectionGcpHasBeenCreated = "A new private link attachment connection has been just created"

	privateLinkAttachmentConnectionGcpUrlPath       = "/networking/v1/private-link-attachment-connections"
	privateLinkAttachmentConnectionGcpResourceLabel = "confluent_private_link_attachment_connection.main"
)

func TestAccPrivateLinkAttachmentConnectionGcp(t *testing.T) {
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

	createPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/create_gcp_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionGcpUrlPath)).
		InScenario(privateLinkAttachmentConnectionGcpResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentConnectionGcpHasBeenCreated).
		WillReturn(
			string(createPlattResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_gcp_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionGcpReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionGcpResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentConnectionGcpHasBeenCreated).
		WillReturn(
			string(readPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionGcpReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionGcpResourceScenarioName).
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
				Config: testAccCheckResourcePrivateLinkAttachmentConnectionGcpWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "id", "plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment-connection=plattc-xyzuvw1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "display_name", "prod-gcp-us-central1-connection"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "gcp.0.private_service_connect_connection_id", "1234567891234"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionGcpResourceLabel, "private_link_attachment.0.id", "platt-abcdef"),
				),
			},
		},
	})
}

func testAccCheckResourcePrivateLinkAttachmentConnectionGcpWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource "confluent_private_link_attachment_connection" "main" {
	    display_name = "prod-gcp-us-central1-connection"
		environment {
			id = "env-12345"
		}
		gcp {
			private_service_connect_connection_id = "1234567891234"
		}
		private_link_attachment {
			id = "platt-abcdef"
		}
	}
	`, mockServerUrl)
}
