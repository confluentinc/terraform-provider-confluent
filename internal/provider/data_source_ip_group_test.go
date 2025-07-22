package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	ipGroupDataSourceScenarioName = "confluent_ip_group Data Source Lifecycle"
)

func TestAccDataSourceGroup(t *testing.T) {
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

	readAwsPeeringGatewayResponse, _ := os.ReadFile("../testdata/ip_group/read_created_ip_group.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-groups/%s", testIPGroupID))).
		InScenario(ipGroupDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsPeeringGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullIPGroupResourceLabel := fmt.Sprintf("data.confluent_ip_group.%s", testIPGroupResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIPGroup(mockServerUrl, testIPGroupID, testIPGroupResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupExists(fullIPGroupResourceLabel),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramId, testIPGroupID),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramGroupName, testIPGroupName),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, "cidr_blocks.#", strconv.Itoa(len(testIPGroupCidrBlocks))),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, "cidr_blocks.0", testIPGroupCidrBlocks[0]),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, "cidr_blocks.1", testIPGroupCidrBlocks[1]),
				),
			},
		},
	})
}

func testAccCheckDataSourceIPGroup(mockServerUrl, resourceId, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_ip_group" "%s" {
	  id = "%s"
	}
	`, mockServerUrl, resourceName, resourceId)
}
