// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

var awsTransitGatewayAttachmentRoutes = []string{
	"192.168.0.0/16",
	"172.16.0.0/12",
	"100.64.0.0/10",
	"10.0.0.0/8",
}

var awsTransitGatewayAttachmentUrlPath = fmt.Sprintf("/networking/v1/transit-gateway-attachments/%s", awsTransitGatewayAttachmentId)

func TestAccAwsTransitGatewayAttachmentAccess(t *testing.T) {
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
	createAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/create_transit_gateway_attachment.json")
	createAwsTransitGatewayAttachmentStub := wiremock.Post(wiremock.URLPathEqualTo("/networking/v1/transit-gateway-attachments")).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAwsTransitGatewayAttachmentIsProvisioning).
		WillReturn(
			string(createAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	if err := wiremockClient.StubFor(createAwsTransitGatewayAttachmentStub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readProvisioningAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_provisioning_transit_gateway_attachment.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsTransitGatewayAttachmentIsProvisioning).
		WillSetStateTo(scenarioStateAwsTransitGatewayAttachmentHasBeenCreated).
		WillReturn(
			string(readProvisioningAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readCreatedAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_created_transit_gateway_attachment.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsTransitGatewayAttachmentHasBeenCreated).
		WillReturn(
			string(readCreatedAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	deleteAwsTransitGatewayAttachmentStub := wiremock.Delete(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsTransitGatewayAttachmentHasBeenCreated).
		WillSetStateTo(scenarioStateAwsTransitGatewayAttachmentIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	if err := wiremockClient.StubFor(deleteAwsTransitGatewayAttachmentStub); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readDeprovisioningAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_deprovisioning_transit_gateway_attachment.json")

	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsTransitGatewayAttachmentIsDeprovisioning).
		WillSetStateTo(scenarioStateAwsTransitGatewayAttachmentHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	readDeletedAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_deleted_transit_gateway_attachment.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(awsTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(scenarioStateAwsTransitGatewayAttachmentHasBeenDeleted).
		WillReturn(
			string(readDeletedAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		)); err != nil {
		t.Errorf("StubFor failed: %v", err)
	}

	awsTransitGatewayAttachmentDisplayName := "prod-tgw-use1"
	awsTransitGatewayAttachmentResourceLabel := "test"
	fullAwsTransitGatewayAttachmentResourceLabel := fmt.Sprintf("confluent_transit_gateway_attachment.%s", awsTransitGatewayAttachmentResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckAwsTransitGatewayAttachmentDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsTransitGatewayAttachmentConfig(mockServerUrl, awsTransitGatewayAttachmentDisplayName, awsTransitGatewayAttachmentResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsTransitGatewayAttachmentExists(fullAwsTransitGatewayAttachmentResourceLabel),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "id", awsTransitGatewayAttachmentId),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "display_name", transitGatewayAttachmentDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.ram_resource_share_arn", awsTransitGatewayAttachmentRamResourceShareArn),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.transit_gateway_id", awsTransitGatewayAttachmentTransitGatewayId),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.routes.#", "4"),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.routes.0", awsTransitGatewayAttachmentRoutes[0]),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.routes.1", awsTransitGatewayAttachmentRoutes[1]),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.routes.2", awsTransitGatewayAttachmentRoutes[2]),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.routes.3", awsTransitGatewayAttachmentRoutes[3]),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "aws.0.transit_gateway_attachment_id", awsTransitGatewayAttachmentTgwaId),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "environment.0.id", awsTransitGatewayAttachmentEnvironmentId),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullAwsTransitGatewayAttachmentResourceLabel, "network.0.id", awsTransitGatewayAttachmentNetworkId),
				),
			},
			{
				ResourceName:      fullAwsTransitGatewayAttachmentResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					awsTransitGatewayAttachmentId := resources[fullAwsTransitGatewayAttachmentResourceLabel].Primary.ID
					environmentId := resources[fullAwsTransitGatewayAttachmentResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + awsTransitGatewayAttachmentId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createAwsTransitGatewayAttachmentStub, fmt.Sprintf("POST %s", awsTransitGatewayAttachmentUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsTransitGatewayAttachmentStub, fmt.Sprintf("DELETE %s?environment=%s", awsTransitGatewayAttachmentUrlPath, awsTransitGatewayAttachmentEnvironmentId), expectedCountOne)
}

func testAccCheckAwsTransitGatewayAttachmentDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each aws transit  gateway attachmentis destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_transit_gateway_attachment" {
			continue
		}
		deletedTransitGatewayAttachmentId := rs.Primary.ID
		req := c.networkingV1Client.TransitGatewayAttachmentsNetworkingV1Api.GetNetworkingV1TransitGatewayAttachment(c.networkingV1ApiContext(context.Background()), deletedTransitGatewayAttachmentId).Environment(awsTransitGatewayAttachmentEnvironmentId)
		deletedTransitGatewayAttachment, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedTransitGatewayAttachment.Id != nil {
			// Otherwise return the error
			if *deletedTransitGatewayAttachment.Id == rs.Primary.ID {
				return fmt.Errorf("aws transit  gateway attachment(%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAwsTransitGatewayAttachmentConfig(mockServerUrl, displayName, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_transit_gateway_attachment" "%s" {
        display_name = "%s"
	    aws {
		  ram_resource_share_arn = "%s"
          transit_gateway_id = "%s"
          routes = [%q, %q, %q, %q]
 		}
		environment {
		  id = "%s"
	    }
		network {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, displayName, awsTransitGatewayAttachmentRamResourceShareArn, awsTransitGatewayAttachmentTransitGatewayId,
		awsTransitGatewayAttachmentRoutes[0], awsTransitGatewayAttachmentRoutes[1], awsTransitGatewayAttachmentRoutes[2], awsTransitGatewayAttachmentRoutes[3],
		awsTransitGatewayAttachmentEnvironmentId, awsTransitGatewayAttachmentNetworkId)
}

func testAccCheckAwsTransitGatewayAttachmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("%s AWS Transit Gateway Attachment has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Aws Transit Gateway Attachment", n)
		}

		return nil
	}
}
