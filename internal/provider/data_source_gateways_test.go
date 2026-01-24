// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	gatewaysDataSourceScenarioName = "confluent_gateways Data Source Lifecycle"
	gatewaysResourceLabel          = "test_gateways_label"
)

func TestAccDataSourceGateways(t *testing.T) {
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

	readGatewaysResponse, _ := os.ReadFile("../testdata/gateway/list_gateways.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways")).
		WithQueryParam("environment", wiremock.EqualTo("env-abc123")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listGatewaysPageSize))).
		InScenario(gatewaysDataSourceScenarioName).
		WillReturn(
			string(readGatewaysResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullGatewaysDataSourceLabel := fmt.Sprintf("data.confluent_gateways.%s", gatewaysResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateways(mockServerUrl, gatewaysResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewaysExists(fullGatewaysDataSourceLabel),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.#", "3"),

					// First gateway - AWS Peering
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_peering_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_peering_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_ingress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_egress_private_link_gateway.#", "0"),

					// Second gateway - AWS Ingress Private Link
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.id", "gw-ingress123"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.display_name", "prod-ingress-gateway"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.aws_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.aws_ingress_private_link_gateway.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.aws_ingress_private_link_gateway.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.1.aws_egress_private_link_gateway.#", "0"),

					// Third gateway - AWS Egress Private Link
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.id", "gw-def456"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.display_name", "prod-egress-gateway"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.aws_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.aws_egress_private_link_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.aws_egress_private_link_gateway.0.principal_arn", "arn:aws:iam::123456789012:role"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.2.aws_ingress_private_link_gateway.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceGatewaysWithFilters(t *testing.T) {
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

	readGatewaysFilteredResponse, _ := os.ReadFile("../testdata/gateway/list_gateways_filtered.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways")).
		WithQueryParam("environment", wiremock.EqualTo("env-abc123")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listGatewaysPageSize))).
		WithQueryParam("gateway_type", wiremock.EqualTo("AwsIngressPrivateLink")).
		WithQueryParam("status.phase", wiremock.EqualTo("READY")).
		InScenario(gatewaysDataSourceScenarioName).
		WillReturn(
			string(readGatewaysFilteredResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullGatewaysDataSourceLabel := fmt.Sprintf("data.confluent_gateways.%s", gatewaysResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGatewaysWithFilters(mockServerUrl, gatewaysResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewaysExists(fullGatewaysDataSourceLabel),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.#", "1"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.id", "gw-ingress123"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.display_name", "prod-ingress-gateway"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_ingress_private_link_gateway.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(fullGatewaysDataSourceLabel, "gateways.0.aws_ingress_private_link_gateway.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
				),
			},
		},
	})
}

func testAccCheckDataSourceGateways(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_gateways" "%s" {
	  environment {
	    id = "env-abc123"
	  }
	}
	`, mockServerUrl, label)
}

func testAccCheckDataSourceGatewaysWithFilters(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_gateways" "%s" {
	  environment {
	    id = "env-abc123"
	  }
	  filter {
	    gateway_type = ["AwsIngressPrivateLink"]
	    phase        = ["READY"]
	  }
	}
	`, mockServerUrl, label)
}

func testAccCheckGatewaysExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s gateways has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s gateways", resourceName)
		}

		return nil
	}
}
