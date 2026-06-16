package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccFlinkComputePoolConfigDataSourceLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	flinkComputePoolConfigResourceLabel := "test_live_compute_pool_config_resource"
	flinkComputePoolConfigDataSourceLabel := "test_live_compute_pool_config_data_source"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkComputePoolConfigDataSourceLiveConfig(endpoint, flinkComputePoolConfigResourceLabel, flinkComputePoolConfigDataSourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					// Check the resource was created
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool_config.%s", flinkComputePoolConfigResourceLabel), "default_compute_pool_enabled", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool_config.%s", flinkComputePoolConfigResourceLabel), "default_max_cfu", "10"),

					// Check the data source can find it
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_flink_compute_pool_config.%s", flinkComputePoolConfigDataSourceLabel), "default_compute_pool_enabled",
						fmt.Sprintf("confluent_flink_compute_pool_config.%s", flinkComputePoolConfigResourceLabel), "default_compute_pool_enabled",
					),
					resource.TestCheckResourceAttrPair(
						fmt.Sprintf("data.confluent_flink_compute_pool_config.%s", flinkComputePoolConfigDataSourceLabel), "default_max_cfu",
						fmt.Sprintf("confluent_flink_compute_pool_config.%s", flinkComputePoolConfigResourceLabel), "default_max_cfu",
					),
				),
			},
		},
	})
}

func testAccCheckFlinkComputePoolConfigDataSourceLiveConfig(endpoint, computePoolConfigResourceLabel, computePoolConfigDataSourceLabel, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_flink_compute_pool_config" "%s" {
	      default_compute_pool_enabled = true
          default_max_cfu = 10
	}

	data "confluent_flink_compute_pool_config" "%s" {
		id = "org123"
	}
	`, endpoint, apiKey, apiSecret, computePoolConfigResourceLabel, computePoolConfigDataSourceLabel)
}
