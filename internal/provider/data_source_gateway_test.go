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
	awsPeeringGatewayScenarioName             = "confluent_gateway AWS Peering Gateway Spec Data Source Lifecycle"
	awsEgressPrivateLinkGatewayScenarioName   = "confluent_gateway AWS Egress Private Link Gateway Spec Data Source Lifecycle"
	azurePeeringGatewayScenarioName           = "confluent_gateway Azure Peering Gateway Spec Data Source Lifecycle"
	azureEgressPrivateLinkGatewayScenarioName = "confluent_gateway Azure Egress Private Link Gateway Spec Data Source Lifecycle"
)

func TestAccDataSourceGatewayAwsPeeringGatewaySpec(t *testing.T) {
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

	readAwsPeeringGatewayResponse, _ := os.ReadFile("../testdata/gateway/read_aws_peering_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways/gw-abc123")).
		InScenario(awsPeeringGatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsPeeringGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	gatewayResourceName := "aws_peering_gateway"
	fullGatewayResourceName := fmt.Sprintf("data.confluent_gateway.%s", gatewayResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateway(mockServerUrl, "gw-abc123", gatewayResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewayExists(fullGatewayResourceName),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "id", "gw-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.0.region", "us-east-2"),
				),
			},
		},
	})
}

func TestAccDataSourceGatewayAwsEgressPrivateLinkGatewaySpec(t *testing.T) {
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

	readAwsEgressPrivateLinkGatewayResponse, _ := os.ReadFile("../testdata/gateway/read_aws_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways/gw-def456")).
		InScenario(awsEgressPrivateLinkGatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsEgressPrivateLinkGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	gatewayResourceName := "aws_egress_private_link_gateway"
	fullGatewayResourceName := fmt.Sprintf("data.confluent_gateway.%s", gatewayResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateway(mockServerUrl, "gw-def456", gatewayResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewayExists(fullGatewayResourceName),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "id", "gw-def456"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.0.principal_arn", "arn:aws:iam::123456789012:role"),
				),
			},
		},
	})
}

func TestAccDataSourceGatewayAwsPrivateNetworkInterfaceGatewaySpec(t *testing.T) {
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

	readAwsEgressPrivateLinkGatewayResponse, _ := os.ReadFile("../testdata/gateway/read_aws_private_network_interface_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways/gw-abc789")).
		InScenario(awsEgressPrivateLinkGatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsEgressPrivateLinkGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	gatewayResourceName := "aws_private_network_interface_gateway"
	fullGatewayResourceName := fmt.Sprintf("data.confluent_gateway.%s", gatewayResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateway(mockServerUrl, "gw-abc789", gatewayResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewayExists(fullGatewayResourceName),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "id", "gw-abc789"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.0.zones.#", "2"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.0.zones.0", "us-east-2a"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.0.zones.1", "us-east-2b"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.0.account", "000000000000"),
				),
			},
		},
	})
}

func TestAccDataSourceGatewayAzurePeeringGatewaySpec(t *testing.T) {
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

	readAzurePeeringGatewayResponse, _ := os.ReadFile("../testdata/gateway/read_azure_peering_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways/gw-def123")).
		InScenario(azurePeeringGatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAzurePeeringGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	gatewayResourceName := "azure_peering_gateway"
	fullGatewayResourceName := fmt.Sprintf("data.confluent_gateway.%s", gatewayResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateway(mockServerUrl, "gw-def123", gatewayResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewayExists(fullGatewayResourceName),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "id", "gw-def123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.0.region", "eastus2"),
				),
			},
		},
	})
}

func TestAccDataSourceGatewayAzureEgressPrivateLinkGatewaySpec(t *testing.T) {
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

	readAzureEgressPrivateLinkGatewayResponse, _ := os.ReadFile("../testdata/gateway/read_azure_egress_private_link_gateway.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/gateways/gw-abc456")).
		InScenario(azureEgressPrivateLinkGatewayScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAzureEgressPrivateLinkGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	gatewayResourceName := "azure_egress_private_link_gateway"
	fullGatewayResourceName := fmt.Sprintf("data.confluent_gateway.%s", gatewayResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGateway(mockServerUrl, "gw-abc456", gatewayResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGatewayExists(fullGatewayResourceName),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "id", "gw-abc456"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "display_name", "prod-gateway"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.#", "1"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_peering_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_egress_private_link_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "aws_private_network_interface_gateway.#", "0"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.0.region", "eastus"),
					resource.TestCheckResourceAttr(fullGatewayResourceName, "azure_egress_private_link_gateway.0.subscription", "aa000000-a000-0a00-00aa-0000aaa0a0a0"),
				),
			},
		},
	})
}

func testAccCheckDataSourceGateway(mockServerUrl, resourceId, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_gateway" "%s" {
	  id = "%s"
	  environment {
	    id = "env-abc123"
	  }
	}
	`, mockServerUrl, resourceName, resourceId)
}

func testAccCheckGatewayExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s gateway has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s gateway", resourceName)
		}

		return nil
	}
}
