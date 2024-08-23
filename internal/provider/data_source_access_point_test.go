package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	AwsEgressAccessPointDataSourceScenarioName   = "confluent_access_point AWS Egress Private Link Endpoint Data Source Lifecycle"
	AzureEgressAccessPointDataSourceScenarioName = "confluent_access_point Azure Egress Private Link Endpoint Data Source Lifecycle"
)

func TestAccDataSourceAccessPointAwsEgressPrivateLinkEndpoint(t *testing.T) {
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

	readAwsEgressAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_aws_egress_ap.json")
	readAccessPointStub := wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/access-points/ap-abc123")).
		InScenario(AwsEgressAccessPointDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsEgressAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readAccessPointStub)

	accessPointResourceName := "aws_egress_private_link_endpoint_access_point"
	fullAccessPointResourceName := fmt.Sprintf("data.confluent_access_point.%s", accessPointResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAccessPoint(mockServerUrl, "ap-abc123", accessPointResourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "gateway.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
		},
	})

}

func TestAccDataSourceAccessPointAzureEgressPrivateLinkEndpoint(t *testing.T) {
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

	readAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_azure_egress_ap.json")
	readAccessPointStub := wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/access-points/ap-def456")).
		InScenario(AzureEgressAccessPointDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readAccessPointStub)

	accessPointResourceName := "azure_egress_private_link_endpoint_access_point"
	fullAccessPointResourceName := fmt.Sprintf("data.confluent_access_point.%s", accessPointResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAccessPoint(mockServerUrl, "ap-def456", accessPointResourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "id", "ap-def456"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "gateway.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "aws_egress_private_link_endpoint.#", "0"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_link_service_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/s-abcde/providers/Microsoft.Network/privateLinkServices/pls-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_link_subresource_name", "sqlServer"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_resource_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/testvpc/providers/Microsoft.Network/privateEndpoints/pe-plt-abcdef-az3"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_domain", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_ip_address", "10.2.0.68"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.#", "2"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.0", "dbname.database.windows.net"),
					resource.TestCheckResourceAttr(fullAccessPointResourceName, "azure_egress_private_link_endpoint.0.private_endpoint_custom_dns_config_domains.1", "dbname-region.database.windows.net"),
				),
			},
		},
	})

}

func testAccCheckDataSourceAccessPoint(mockServerUrl, resourceId, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_access_point" "%s" {
      id = "%s"
	  environment {
		id = "env-abc123"
	  }
	}
	`, mockServerUrl, resourceName, resourceId)
}
