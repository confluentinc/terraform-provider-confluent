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
	privateLinkAttachmentGcpResourceScenarioName        = "confluent_private_link_attachment Resource Lifecycle"
	scenarioStatePrivateLinkAttachmentGcpHasBeenCreated = "A new private link attachment has been just created"

	privateLinkAttachmentGcpUrlPath       = "/networking/v1/private-link-attachments"
	privateLinkAttachmentGcpResourceLabel = "confluent_private_link_attachment.main"
)

func TestAccPrivateLinkAttachmentGcp(t *testing.T) {
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

	createPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/create_gcp_platt.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(privateLinkAttachmentGcpUrlPath)).
		InScenario(privateLinkAttachmentGcpResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentGcpHasBeenCreated).
		WillReturn(
			string(createPlattResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/read_gcp_platt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentGcpReadUrlPath)).
		InScenario(privateLinkAttachmentGcpResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentGcpHasBeenCreated).
		WillReturn(
			string(readPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(privateLinkAttachmentGcpReadUrlPath)).
		InScenario(privateLinkAttachmentGcpResourceScenarioName).
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
				Config: testAccCheckResourcePrivateLinkAttachmentGcpWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "id", "platt-abcdef"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/private-link-attachment=platt-abcdef"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "display_name", "prod-gcp-us-central1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "environment.0.id", "env-12345"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "region", "us-central1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "gcp.#", "1"),
					resource.TestCheckResourceAttr(privateLinkAttachmentGcpResourceLabel, "gcp.0.private_service_connect_service_attachment", "projects/project/regions/us-central1/serviceAttachments/plattg-abcd123-service-attachment"),
				),
			},
		},
	})
}

func testAccCheckResourcePrivateLinkAttachmentGcpWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource "confluent_private_link_attachment" "main" {
	    cloud = "GCP"
	    region = "us-central1"
	    display_name = "prod-gcp-us-central1"
	    environment {
		    id = "env-12345"
	    }
	}
	`, mockServerUrl)
}
