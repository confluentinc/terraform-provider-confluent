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
	ipFilterDataSourceScenarioName = "confluent_ip_filter Data Source Lifecycle"
)

func TestAccDataSourceFilter(t *testing.T) {
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

	readAwsPeeringGatewayResponse, _ := os.ReadFile("../testdata/ip_filter/read_created_ip_filter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/ip-filters/%s", testIPFilterID))).
		InScenario(ipFilterDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsPeeringGatewayResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullIPGroupResourceLabel := fmt.Sprintf("data.confluent_ip_filter.%s", testIpFilterResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceIPFilter(mockServerUrl, testIPFilterID, testIpFilterResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIPGroupExists(fullIPGroupResourceLabel),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramId, testIPFilterID),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramFilterName, testIpFilterName),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramResourceGroup, testIpFilterResourceGroup),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, paramResourceScope, testIpFilterResourceScope),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.#", paramOperationGroups), strconv.Itoa(len(testIpFilterOperationGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramOperationGroups), testIpFilterOperationGroups[0]),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramOperationGroups), testIpFilterOperationGroups[1]),
					resource.TestCheckResourceAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.#", paramIPGroups), strconv.Itoa(len(testIpFilterIpGroups))),
					resource.TestCheckTypeSetElemAttr(fullIPGroupResourceLabel, fmt.Sprintf("%s.*", paramIPGroups), testIpFilterIpGroups[0]),
				),
			},
		},
	})
}

func testAccCheckDataSourceIPFilter(mockServerUrl, resourceId, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_ip_filter" "%s" {
	  id = "%s"
	}
	`, mockServerUrl, resourceName, resourceId)
}
