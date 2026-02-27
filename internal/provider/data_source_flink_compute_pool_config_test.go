package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	flinkComputePoolDataSourceConfigScenarioName = "confluent_flink_compute_pool_config Data Source Lifecycle"
)

var fullFlinkComputePoolConfigDataSourceLabel = fmt.Sprintf("data.confluent_flink_compute_pool_config.test")

func TestAccDataFlinkComputePoolConfigSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := mockSchemaTestServerUrl
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedComputePoolConfigResponse, _ := ioutil.ReadFile("../testdata/flink_compute_pool_config/read_created_compute_pool_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolDataSourceConfigScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedComputePoolConfigResponse),
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
				Config: testAccCheckFlinkComputePoolDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolConfigExists(fullFlinkComputePoolConfigDataSourceLabel),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigDataSourceLabel, "default_compute_pool_enabled", "true"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigDataSourceLabel, "default_max_cfu", "10"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigDataSourceLabel, "api_version", "fcpm/v2"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigDataSourceLabel, "kind", "OrgComputePoolConfig"),
				),
			},
		},
	})
}

func testAccCheckFlinkComputePoolDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_flink_compute_pool_config" "test" {
    	id = "7c210ed4-6e1e-4355-abf9-b25e25a8b25a"
	}
	`, confluentCloudBaseUrl)
}

func testAccCheckComputePoolConfigExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s schema has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s schema", n)
		}

		return nil
	}
}
