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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceTransitGatewayAttachmentScenarioName = "confluent_transit_gateway_attachment Data Source Lifecycle"
	transitGatewayAttachmentDataSourceLabel        = "example"
	transitGatewayAttachmentDataSourceDisplayName  = "prod-tgw-use1"
)

var fullTransitGatewayAttachmentDataSourceLabel = fmt.Sprintf("data.confluent_transit_gateway_attachment.%s", transitGatewayAttachmentDataSourceLabel)

func TestAccDataSourceTransitGatewayAttachment(t *testing.T) {
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

	readCreatedAwsTransitGatewayAttachmentResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_created_transit_gateway_attachment.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsTransitGatewayAttachmentUrlPath)).
		InScenario(dataSourceTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsTransitGatewayAttachmentResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readTransitGatewayAttachmentsResponse, _ := ioutil.ReadFile("../testdata/transit_gateway_attachment/aws/read_transit_gateway_attachments.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/transit-gateway-attachments")).
		InScenario(dataSourceTransitGatewayAttachmentScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsTransitGatewayAttachmentEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readTransitGatewayAttachmentsResponse),
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
				Config: testAccCheckDataSourceTransitGatewayAttachmentWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsTransitGatewayAttachmentExists(fullTransitGatewayAttachmentDataSourceLabel),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "id", awsTransitGatewayAttachmentId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "display_name", transitGatewayAttachmentDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.ram_resource_share_arn", awsTransitGatewayAttachmentRamResourceShareArn),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.transit_gateway_id", awsTransitGatewayAttachmentTransitGatewayId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.enable_custom_routes", awsTransitGatewayAttachmentEnableCustomRoutes),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.#", "4"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.0", awsTransitGatewayAttachmentRoutes[0]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.1", awsTransitGatewayAttachmentRoutes[1]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.2", awsTransitGatewayAttachmentRoutes[2]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.3", awsTransitGatewayAttachmentRoutes[3]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.transit_gateway_attachment_id", awsTransitGatewayAttachmentTgwaId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "environment.0.id", awsTransitGatewayAttachmentEnvironmentId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "network.0.id", awsTransitGatewayAttachmentNetworkId),
				),
			},
			{
				Config: testAccCheckDataSourceTransitGatewayAttachmentWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsTransitGatewayAttachmentExists(fullTransitGatewayAttachmentDataSourceLabel),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "id", awsTransitGatewayAttachmentId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "display_name", transitGatewayAttachmentDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.ram_resource_share_arn", awsTransitGatewayAttachmentRamResourceShareArn),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.transit_gateway_id", awsTransitGatewayAttachmentTransitGatewayId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.enable_custom_routes", awsTransitGatewayAttachmentEnableCustomRoutes),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.#", "4"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.0", awsTransitGatewayAttachmentRoutes[0]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.1", awsTransitGatewayAttachmentRoutes[1]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.2", awsTransitGatewayAttachmentRoutes[2]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.routes.3", awsTransitGatewayAttachmentRoutes[3]),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "aws.0.transit_gateway_attachment_id", awsTransitGatewayAttachmentTgwaId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "environment.0.id", awsTransitGatewayAttachmentEnvironmentId),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullTransitGatewayAttachmentDataSourceLabel, "network.0.id", awsTransitGatewayAttachmentNetworkId),
				),
			},
		},
	})
}

func testAccCheckDataSourceTransitGatewayAttachmentWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_transit_gateway_attachment" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, transitGatewayAttachmentDataSourceLabel, transitGatewayAttachmentDataSourceDisplayName, awsTransitGatewayAttachmentEnvironmentId)
}

func testAccCheckDataSourceTransitGatewayAttachmentWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_transit_gateway_attachment" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, transitGatewayAttachmentDataSourceLabel, awsTransitGatewayAttachmentId, awsTransitGatewayAttachmentEnvironmentId)
}
