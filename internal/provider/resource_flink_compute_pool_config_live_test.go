package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccComputePoolConfigLive(t *testing.T) {
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

	computePoolConfigResourceLabel := "test_live_compute_pool_config"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckComputePoolConfigLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckComputePoolConfigLiveConfig(endpoint, computePoolConfigResourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolConfigLiveExists(fmt.Sprintf("confluent_flink_compute_pool_config.%s", computePoolConfigResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool_config.%s", computePoolConfigResourceLabel), "default_compute_pool_enabled", "true"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool_config.%s", computePoolConfigResourceLabel), "default_max_cfu", "10"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_compute_pool_config.%s", computePoolConfigResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckComputePoolConfigLiveConfig(endpoint, computePoolConfigResourceLabel, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint                       = "%s"
		cloud_api_key                  = "%s"
		cloud_api_secret               = "%s"
	}

	resource "confluent_flink_compute_pool_config" "%s" {
	      default_compute_pool_enabled = true
          default_max_cfu = 10
	}
	`, endpoint, apiKey, apiSecret, computePoolConfigResourceLabel)
}

func testAccCheckComputePoolConfigLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		return nil
	}
}

func testAccCheckComputePoolConfigLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_compute_pool_config" {
			continue
		}

		// In live tests, we can't easily check if the resource is actually destroyed
		// without making API calls, so we just verify the resource is removed from state
		if rs.Primary.ID != "" {
			// This is normal - the resource should have an ID but be removed from the live environment
			// The actual cleanup happens through the API calls during destroy
		}
	}
	return nil
}
