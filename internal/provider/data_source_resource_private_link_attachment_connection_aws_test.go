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
	privateLinkAttachmentConnectionAwsDataSourceScenarioName = "confluent_private_link_attachment_connection Data Source Lifecycle"

	privateLinkAttachmentConnectionAwsReadUrlPath     = "/networking/v1/private-link-attachment-connections/plattc-gz20xy"
	privateLinkAttachmentConnectionAwsId              = "plattc-gz20xy"
	privateLinkAttachmentConnectionAwsDataSourceLabel = "data.confluent_private_link_attachment_connection.main"
)

func TestAccDataSourcePrivateLinkAttachmentConnectionAws(t *testing.T) {
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

	readPrivateLinkAttachmentConnectionAwsResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment_connection/read_aws_plattc.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentConnectionAwsReadUrlPath)).
		InScenario(privateLinkAttachmentConnectionAwsDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPrivateLinkAttachmentConnectionAwsResponse),
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
				Config: testAccCheckDataSourcePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "id", "plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-plyvyl/private-link-attachment-connection=plattc-gz20xy"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "display_name", "my_vpc"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "aws.0.vpc_endpoint_id", "vpce-0ed4d51f5d6ef9b6d"),
					resource.TestCheckResourceAttr(privateLinkAttachmentConnectionAwsDataSourceLabel, "private_link_attachment.0.id", "platt-plyvyl"),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourcePrivateLinkAttachmentConnectionAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
  		endpoint = "%s"
 	}

 	data "confluent_private_link_attachment_connection" "main" {
 		id = "%s"
         environment {
 			id = "env-8gv0v5"
 	  	}
 	}
 	`, mockServerUrl, privateLinkAttachmentConnectionAwsId)
}
