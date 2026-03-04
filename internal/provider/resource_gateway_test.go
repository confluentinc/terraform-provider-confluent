// Copyright 2022 Confluent Inc. All Rights Reserved.
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

	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	scenarioStateGatewayIsProvisioning = "The new gateway is provisioning"
	scenarioStateGatewayHasBeenCreated = "The new gateway has been just created"
	scenarioStateGatewayHasBeenUpdated = "The new gateway has been updated"
	GatewayScenarioName                = "confluent_gateway Resource Lifecycle"

	gatewayUrlPath               = "/networking/v1/gateways"
	awsGatewayId                 = "gw-def456"
	awsIngressGatewayId          = "gw-ingress123"
	awsPrivateNetworkInterfaceId = "gw-abc789"
	azureGatewayId               = "gw-abc456"
	azureIngressGatewayId        = "gw-azure-ingress"
	gcpIngressGatewayId          = "gw-gcp-ingress"
	gatewayResourceLabel         = "confluent_gateway.main"
)

func TestAccGatewayAwsEgressPrivateLink(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_aws_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_aws_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_aws_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsGatewayId))).
		InScenario(GatewayScenarioName).
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
				Config: testAccCheckResourceGatewayAwsEgressPrivateLinkConfig(mockServerUrl, "prod-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.0.principal_arn", "arn:aws:iam::123456789012:role"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayAwsEgressPrivateLinkConfig(mockServerUrl, "prod-gateway-new"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway-new"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.0.principal_arn", "arn:aws:iam::123456789012:role"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
		},
	})
}

func TestAccGatewayAwsPrivateNetworkInterface(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_aws_private_network_interface_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsPrivateNetworkInterfaceId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_aws_private_network_interface_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsPrivateNetworkInterfaceId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_aws_private_network_interface_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsPrivateNetworkInterfaceId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsPrivateNetworkInterfaceId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsPrivateNetworkInterfaceId))).
		InScenario(GatewayScenarioName).
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
				Config: testAccCheckResourceGatewayAwsPrivateNetworkInterfaceConfig(mockServerUrl, "prod-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsPrivateNetworkInterfaceId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.#", "2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.0", "us-east-2a"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.1", "us-east-2b"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.account", "000000000000"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayAwsPrivateNetworkInterfaceConfig(mockServerUrl, "prod-gateway-new"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsPrivateNetworkInterfaceId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway-new"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.#", "2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.0", "us-east-2a"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.zones.1", "us-east-2b"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.0.account", "000000000000"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
		},
	})
}

func TestAccGatewayAzure(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_azure_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_azure_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_azure_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureGatewayId))).
		InScenario(GatewayScenarioName).
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
				Config: testAccCheckResourceGatewayAzureEgressPrivateLinkConfig(mockServerUrl, "prod-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", azureGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.0.region", "eastus"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.0.subscription", "aa000000-a000-0a00-00aa-0000aaa0a0a0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayAzureEgressPrivateLinkConfig(mockServerUrl, "prod-gateway-new"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", azureGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gateway-new"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.0.region", "eastus"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.0.subscription", "aa000000-a000-0a00-00aa-0000aaa0a0a0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
				),
			},
		},
	})
}

func testAccCheckResourceGatewayAwsEgressPrivateLinkConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		aws_egress_private_link_gateway {
			region = "us-east-2"
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceGatewayAwsPrivateNetworkInterfaceConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		aws_private_network_interface_gateway {
			region = "us-east-2"
			zones = ["us-east-2a", "us-east-2b"]
		}
	}
	`, mockServerUrl, name)
}

func TestAccGatewayAwsIngressPrivateLink(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_aws_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_aws_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_aws_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, awsIngressGatewayId))).
		InScenario(GatewayScenarioName).
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
				Config: testAccCheckResourceGatewayAwsIngressPrivateLinkConfig(mockServerUrl, "prod-ingress-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-ingress-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayAwsIngressPrivateLinkConfig(mockServerUrl, "prod-ingress-gateway-new"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", awsIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-ingress-gateway-new"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.0.region", "us-west-2"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_ingress_private_link_gateway.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_egress_private_link_gateway.#", "0"),
				),
			},
		},
	})
}

func testAccCheckResourceGatewayAwsIngressPrivateLinkConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		aws_ingress_private_link_gateway {
			region = "us-west-2"
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceGatewayAzureEgressPrivateLinkConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		azure_egress_private_link_gateway {
			region = "eastus"
		}
	}
	`, mockServerUrl, name)
}
func TestAccGatewayAzureIngressPrivateLink(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_azure_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_azure_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_azure_ingress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, azureIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceGatewayAzureIngressPrivateLinkConfig(mockServerUrl, "prod-azure-ingress-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", azureIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-azure-ingress-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.0.region", "centralus"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.0.private_link_service_alias", "plattg-123abc-privatelink.00000000-0000-0000-0000-000000000000.centralus.azure.privatelinkservice"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.0.private_link_service_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/plattg-123abc/providers/Microsoft.Network/privateLinkServices/plattg-123abc-privatelink"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayAzureIngressPrivateLinkConfig(mockServerUrl, "updated-azure-ingress-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", azureIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "updated-azure-ingress-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.0.region", "centralus"),
				),
			},
		},
	})
}

func TestAccGatewayGcpIngressPrivateServiceConnect(t *testing.T) {
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
	createGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/create_gcp_ingress_private_service_connect_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(gatewayUrlPath)).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGatewayIsProvisioning).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, gcpIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayIsProvisioning).
		WillSetStateTo(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(createGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_gcp_ingress_private_service_connect_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, gcpIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillReturn(
			string(readGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedGatewayResponse, _ := ioutil.ReadFile("../testdata/gateway/read_updated_gcp_ingress_private_service_connect_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, gcpIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenCreated).
		WillSetStateTo(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, gcpIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WhenScenarioStateIs(scenarioStateGatewayHasBeenUpdated).
		WillReturn(
			string(readUpdatedGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/%s", gatewayUrlPath, gcpIngressGatewayId))).
		InScenario(GatewayScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceGatewayGcpIngressPrivateServiceConnectConfig(mockServerUrl, "prod-gcp-ingress-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", gcpIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "prod-gcp-ingress-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.0.region", "us-central1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.0.private_service_connect_service_attachment", "projects/traffic-prod/regions/us-central1/serviceAttachments/plattg-abc123-service-attachment"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "azure_ingress_private_link_gateway.#", "0"),
				),
			},
			{
				Config: testAccCheckResourceGatewayGcpIngressPrivateServiceConnectConfig(mockServerUrl, "updated-gcp-ingress-gateway"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(gatewayResourceLabel, "id", gcpIngressGatewayId),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "display_name", "updated-gcp-ingress-gateway"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.#", "1"),
					resource.TestCheckResourceAttr(gatewayResourceLabel, "gcp_ingress_private_service_connect_gateway.0.region", "us-central1"),
				),
			},
		},
	})
}

func testAccCheckResourceGatewayAzureIngressPrivateLinkConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		azure_ingress_private_link_gateway {
			region = "centralus"
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceGatewayGcpIngressPrivateServiceConnectConfig(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_gateway" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		gcp_ingress_private_service_connect_gateway {
			region = "us-central1"
		}
	}
	`, mockServerUrl, name)
}
