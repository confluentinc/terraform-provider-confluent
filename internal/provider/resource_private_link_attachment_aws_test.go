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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

func TestAccPrivateLinkAttachmentAws(t *testing.T) {
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

	createPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/create_aws_platt.json")
	if err := wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(privateLinkAttachmentAwsUrlPath)).
		InScenario(privateLinkAttachmentAwsResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentAwsHasBeenCreated).
		WillReturn(
			string(createPlattResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	readPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/read_aws_platt.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentAwsReadUrlPath)).
		InScenario(privateLinkAttachmentAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentAwsHasBeenCreated).
		WillReturn(
			string(readPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	updatedPlattResponse, _ := ioutil.ReadFile("../testdata/private_link_attachment/read_updated_aws_platt.json")
	if err := wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(privateLinkAttachmentAwsReadUrlPath)).
		InScenario(privateLinkAttachmentAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentAwsHasBeenCreated).
		WillSetStateTo(scenarioStatePrivateLinkAttachmentAwsHasBeenUpdated).
		WillReturn(
			string(updatedPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(privateLinkAttachmentAwsReadUrlPath)).
		InScenario(privateLinkAttachmentAwsResourceScenarioName).
		WhenScenarioStateIs(scenarioStatePrivateLinkAttachmentAwsHasBeenUpdated).
		WillReturn(
			string(updatedPlattResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	if err := wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(privateLinkAttachmentAwsReadUrlPath)).
		InScenario(privateLinkAttachmentAwsResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourcePrivateLinkAttachmentAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "id", "platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "display_name", "staging-aws-us-west"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "region", "us-west-2"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "dns_domain", "pr1jy6.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "aws.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-0d3be37e21aaaaaa"),
				),
			},
			{
				Config: testAccCheckResourceUpdatePrivateLinkAttachmentAwsWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "id", "platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "resource_name", "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-8gv0v5/private-link-attachment=platt-61ovvd"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "display_name", "staging-aws-us-west-updated"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "environment.0.id", "env-8gv0v5"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "region", "us-west-2"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "dns_domain", "pr1jy6.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(privateLinkAttachmentAwsResourceLabel, "aws.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-0d3be37e21aaaaaa"),
				),
			},
		},
	})
}

func testAccCheckResourcePrivateLinkAttachmentAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource confluent_private_link_attachment main {
	    cloud = "AWS"
	    region = "us-west-2"
	    display_name = "staging-aws-us-west"
	    environment {
		    id = "env-8gv0v5"
	    }
	}
	`, mockServerUrl)
}

func testAccCheckResourceUpdatePrivateLinkAttachmentAwsWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

    resource confluent_private_link_attachment main {
	    cloud = "AWS"
	    region = "us-west-2"
	    display_name = "staging-aws-us-west-updated"
	    environment {
		    id = "env-8gv0v5"
	    }
	}
	`, mockServerUrl)
}
