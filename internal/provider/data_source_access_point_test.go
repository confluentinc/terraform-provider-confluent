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

const accessPointDataSourceScenarioName = "confluent_access_point Data Source Lifecycle"

var accessPointDataSourceLabel = fmt.Sprintf("data.%s", accessPointResourceLabel)

func TestAccDataSourceAccessPoint(t *testing.T) {
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

	readAccessPointResponse, _ := os.ReadFile("../testdata/network_access_point/read_created_ap.json")
	readAccessPointStub := wiremock.Get(wiremock.URLPathEqualTo(accessPointReadUrlPath)).
		InScenario(accessPointDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAccessPointResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readAccessPointStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAccessPoint(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "id", "ap-abc123"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "display_name", "prod-ap-1"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "aws_egress_private_link_endpoint.#", "1"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_service_name", "com.amazonaws.vpce.us-west-2.vpce-svc-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_id", "vpce-00000000000000000"),
					resource.TestCheckResourceAttr(accessPointDataSourceLabel, "aws_egress_private_link_endpoint.0.vpc_endpoint_dns_name", "*.vpce-00000000000000000-abcd1234.s3.us-west-2.vpce.amazonaws.com"),
				),
			},
		},
	})

}

func testAccCheckDataSourceAccessPoint(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_access_point" "main" {
      id = "ap-abc123"
	  environment {
		id = "env-abc123"
	  }
	}
	`, mockServerUrl)
}
