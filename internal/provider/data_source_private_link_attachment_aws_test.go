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
	privateLinkAttachmentAwsDataSourceScenarioName = "confluent_private_link_attachment Data Source Lifecycle"

	privateLinkAttachmentAwsReadUrlPath     = "/networking/v1/private-link-attachments/platt-61ovvd"
	privateLinkAttachmentAwsId              = "platt-61ovvd"
	privateLinkAttachmentAwsDataSourceLabel = "data.confluent_private_link_attachment.main"
)

func TestAccDataSourcePrivateLinkAttachmentAws(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	readPrivateLinkAttachmentAwsResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/read_aws_platt.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentAwsReadUrlPath)).
		InScenario(privateLinkAttachmentAwsDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPrivateLinkAttachmentAwsResponse),
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
				Config: testAccCheckDataSourcePrivateLinkAttachmentAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "id", "platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "display_name", "staging-aws-us-west"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "region", "us-west-2"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "dns_domain", "pr1jy6.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsDataSourceLabel, "aws.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-0d3be37e21aaaaaa"),
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

func testAccCheckDataSourcePrivateLinkAttachmentAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
  		endpoint = "%s"
 	}

 	data "confluent_private_link_attachment" "main" {
 		id = "%s"
         environment {
 			id = "env-8gv0v5"
 	  	}
 	}
 	`, mockServerUrl, privateLinkAttachmentAwsId)
}
