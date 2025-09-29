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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateAccessPointIsProvisioning   = "The new access point is provisioning"
	scenarioStateAccessPointHasBeenCreated   = "The new access point has been just created"
	scenarioStateAccessPointHasBeenUpdated   = "The new access point has been updated"
	scenarioStateAccessPointIsDeprovisioning = "The new access point is deprovisioning"
	scenarioStateAccessPointHasBeenDeleted   = "The new access point's deletion has been just completed"

	awsEgressAccessPointScenarioName                  = "confluent_access_point Aws Egress Private Link Endpoint Resource Lifecycle"
	awsPrivateNetworkInterfaceAccessPointScenarioName = "confluent_access_point Aws Private Network Interface Endpoint Resource Lifecycle"
	azureEgressAccessPointScenarioName                = "confluent_access_point Azure Egress Private Link Endpoint Resource Lifecycle"
	gcpEgressAccessPointScenarioName                  = "confluent_access_point Gcp Egress Private Link Endpoint Resource Lifecycle"

	accessPointUrlPath       = "/networking/v1/access-points"
	accessPointResourceLabel = "confluent_access_point.main"
)

func TestAccAccessPointAwsEgressPrivateLinkEndpoint(t *testing.T) {
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

	createAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/create_aws_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(accessPointUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAccessPointIsProvisioning).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	accessPointReadUrlPath := fmt.Sprintf("%s/ap-abc123", accessPointUrlPath)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsProvisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_aws_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(readCreatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/update_aws_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillSetStateTo(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillSetStateTo(scenarioStateAccessPointIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeprovisioningAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deprovisioning_aws_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsDeprovisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deleted_aws_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeletedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceAccessPointAwsEgressWithIdSet(mockServerUrl, "prod-ap-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
			{
				Config: testAccCheckResourceAccessPointAwsEgressWithIdSet(mockServerUrl, "prod-ap-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
			{
				ResourceName: accessPointResourceLabel,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					accessPointId := resources[accessPointResourceLabel].Primary.ID
					environmentId := resources[accessPointResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + accessPointId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAccessPointAwsPrivateNetworkInterface(t *testing.T) {
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

	createAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/create_aws_private_network_interface_ap.json") // private network interface has no status, so we can use the same json file for create and read
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(accessPointUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAccessPointIsProvisioning).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	accessPointReadUrlPath := fmt.Sprintf("%s/ap-abc456", accessPointUrlPath)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsProvisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_aws_private_network_interface_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(readCreatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/update_aws_private_network_interface_ap.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillSetStateTo(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillSetStateTo(scenarioStateAccessPointIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeprovisioningAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deprovisioning_aws_private_network_interface_ap.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsDeprovisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deleted_aws_private_network_interface_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(awsPrivateNetworkInterfaceAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeletedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceAccessPointAwsPrivateNetworkInterfaceWithIdSet(mockServerUrl, "prod-ap-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc456"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.#", "6"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.0", "eni-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.1", "eni-00000000000000001"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.account", "000000000000"),
				),
			},
			{
				Config: testAccCheckResourceAccessPointAwsPrivateNetworkInterfaceWithIdSet(mockServerUrl, "prod-ap-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc456"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.#", "6"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.0", "eni-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.network_interfaces.1", "eni-00000000000000001"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.0.account", "000000000000"),
				),
			},
			{
				ResourceName: accessPointResourceLabel,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					accessPointId := resources[accessPointResourceLabel].Primary.ID
					environmentId := resources[accessPointResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + accessPointId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAccessPointAzureEgressPrivateLinkEndpoint(t *testing.T) {
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

	createAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/create_azure_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(accessPointUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAccessPointIsProvisioning).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	accessPointReadUrlPath := fmt.Sprintf("%s/ap-def456", accessPointUrlPath)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsProvisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_azure_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(readCreatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/update_azure_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillSetStateTo(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillSetStateTo(scenarioStateAccessPointIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeprovisioningAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deprovisioning_azure_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsDeprovisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deleted_azure_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(azureEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeletedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceAccessPointAzureEgressWithIdSet(mockServerUrl, "prod-ap-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-def456"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_link_service_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/s-abcde/providers/Microsoft.Network/privateLinkServices/pls-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_link_subresource_name", "sqlServer"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_domain", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_ip_address", "10.2.0.68"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.#", "2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.0", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.1", "dbname-region.database.windows.net"),
				),
			},
			{
				Config: testAccCheckResourceAccessPointAzureEgressWithIdSet(mockServerUrl, "prod-ap-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-def456"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_link_service_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/s-abcde/providers/Microsoft.Network/privateLinkServices/pls-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_link_subresource_name", "sqlServer"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_domain", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_ip_address", "10.2.0.68"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.#", "2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.0", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.1", "dbname-region.database.windows.net"),
				),
			},
			{
				ResourceName: accessPointResourceLabel,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					accessPointId := resources[accessPointResourceLabel].Primary.ID
					environmentId := resources[accessPointResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + accessPointId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAccessPointGcpEgressPrivateServiceConnectEndpoint(t *testing.T) {
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

	createAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/create_gcp_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(accessPointUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAccessPointIsProvisioning).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	accessPointReadUrlPath := fmt.Sprintf("%s/ap-abc123", accessPointUrlPath)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsProvisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(createAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_gcp_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillReturn(
			string(readCreatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/update_gcp_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenCreated).
		WillSetStateTo(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillReturn(
			string(updatedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenUpdated).
		WillSetStateTo(scenarioStateAccessPointIsDeprovisioning).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	readDeprovisioningAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deprovisioning_gcp_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointIsDeprovisioning).
		WillSetStateTo(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeprovisioningAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_deleted_gcp_egress_ap.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(gcpEgressAccessPointScenarioName).
		WhenScenarioStateIs(scenarioStateAccessPointHasBeenDeleted).
		WillReturn(
			string(readDeletedAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceAccessPointGcpEgressWithIdSet(mockServerUrl, "prod-ap-1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_connection_id", "1234567890987654321"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_ip_address", "10.2.255.255"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_name", "plapstgc493ll4"),
				),
			},
			{
				Config: testAccCheckResourceAccessPointGcpEgressWithIdSet(mockServerUrl, "prod-ap-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointResourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "display_name", "prod-ap-2"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "aws_private_network_interface.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_connection_id", "1234567890987654321"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_ip_address", "10.2.255.255"),
					resource.TestCheckResourceAttr(accessPointResourceLabel, "gcp_egress_private_service_connect_endpoint.0.private_service_connect_endpoint_name", "plapstgc493ll4"),
				),
			},
			{
				ResourceName: accessPointResourceLabel,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					accessPointId := resources[accessPointResourceLabel].Primary.ID
					environmentId := resources[accessPointResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + accessPointId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckResourceAccessPointAwsEgressWithIdSet(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		aws_egress_private_link_endpoint {
			vpc_endpoint_service_name = "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceAccessPointAwsPrivateNetworkInterfaceWithIdSet(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		aws_private_network_interface {
			account = "000000000000"
			network_interfaces = ["eni-00000000000000000", "eni-00000000000000001", "eni-00000000000000002", "eni-00000000000000003", "eni-00000000000000004", "eni-00000000000000005"]
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceAccessPointAzureEgressWithIdSet(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		azure_egress_private_link_endpoint {
			private_link_service_resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/s-abcde/providers/Microsoft.Network/privateLinkServices/pls-plt-abcdef-az3"
			private_link_subresource_name = "sqlServer"
		}
	}
	`, mockServerUrl, name)
}

func testAccCheckResourceAccessPointGcpEgressWithIdSet(mockServerUrl, name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_access_point" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		gateway {
			id = "gw-abc123"
		}
		gcp_egress_private_service_connect_endpoint {
    		private_service_connect_endpoint_target = "ALL_GOOGLE_APIS"
  		}
	}
	`, mockServerUrl, name)
}
