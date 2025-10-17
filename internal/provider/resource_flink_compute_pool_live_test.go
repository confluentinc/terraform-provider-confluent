//go:build live_test && (all || flink)

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
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFlinkComputePoolAWSLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	environmentId := "env-zyg27z"

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if environmentId == "" {
		t.Fatal("LIVE_TEST_ENVIRONMENT_ID must be set for Flink Compute Pool live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	envDisplayName := fmt.Sprintf("tf-live-flink-env-%d", randomSuffix)
	poolDisplayName := fmt.Sprintf("tf-live-flink-pool-aws-%d", randomSuffix)
	poolUpdatedDisplayName := fmt.Sprintf("tf-live-flink-pool-aws-updated-%d", randomSuffix)
	envResourceLabel := "test_live_flink_env"
	poolResourceLabel := "test_live_flink_compute_pool_aws"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkComputePoolLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkComputePoolLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolDisplayName, "AWS", "us-east-2", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkComputePoolLiveExists(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "display_name", poolDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "region", "us-east-2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "max_cfu", "5"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "resource_name"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, poolId), nil
				},
			},
			{
				Config: testAccCheckFlinkComputePoolUpdateLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolUpdatedDisplayName, "AWS", "us-east-2", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkComputePoolLiveExists(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "display_name", poolUpdatedDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "max_cfu", "10"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, poolId), nil
				},
			},
		},
	})
}

func TestAccFlinkComputePoolAzureLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	envDisplayName := fmt.Sprintf("tf-live-flink-env-azure-%d", randomSuffix)
	poolDisplayName := fmt.Sprintf("tf-live-flink-pool-azure-%d", randomSuffix)
	envResourceLabel := "test_live_flink_env_azure"
	poolResourceLabel := "test_live_flink_compute_pool_azure"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkComputePoolLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkComputePoolLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolDisplayName, "AZURE", "eastus2", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkComputePoolLiveExists(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "display_name", poolDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "cloud", "AZURE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "region", "eastus2"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, poolId), nil
				},
			},
		},
	})
}

func TestAccFlinkComputePoolGCPLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	envDisplayName := fmt.Sprintf("tf-live-flink-env-gcp-%d", randomSuffix)
	poolDisplayName := fmt.Sprintf("tf-live-flink-pool-gcp-%d", randomSuffix)
	envResourceLabel := "test_live_flink_env_gcp"
	poolResourceLabel := "test_live_flink_compute_pool_gcp"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkComputePoolLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkComputePoolLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolDisplayName, "GCP", "us-central1", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkComputePoolLiveExists(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "display_name", poolDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "cloud", "GCP"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "region", "us-central1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, poolId), nil
				},
			},
		},
	})
}

func testAccCheckFlinkComputePoolLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Flink Compute Pool is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_compute_pool" {
			continue
		}
		deletedPoolId := rs.Primary.ID
		environmentId := rs.Primary.Attributes["environment.0.id"]
		req := c.fcpmClient.ComputePoolsFcpmV2Api.GetFcpmV2ComputePool(c.fcpmApiContext(context.Background()), deletedPoolId).Environment(environmentId)
		deletedPool, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedPool.Id != nil {
			if *deletedPool.Id == rs.Primary.ID {
				return fmt.Errorf("Flink Compute Pool (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckFlinkComputePoolLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolDisplayName, cloud, region, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
	}

	resource "confluent_flink_compute_pool" "%s" {
		display_name = "%s"
		cloud = "%s"
		region = "%s"
		max_cfu = 5
		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, envResourceLabel, envDisplayName, poolResourceLabel, poolDisplayName, cloud, region, envResourceLabel)
}

func testAccCheckFlinkComputePoolUpdateLiveConfig(endpoint, envResourceLabel, poolResourceLabel, envDisplayName, poolDisplayName, cloud, region, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
	}

	resource "confluent_flink_compute_pool" "%s" {
		display_name = "%s"
		cloud = "%s"
		region = "%s"
		max_cfu = 10
		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, envResourceLabel, envDisplayName, poolResourceLabel, poolDisplayName, cloud, region, envResourceLabel)
}

func testAccCheckFlinkComputePoolLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Flink Compute Pool has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Flink Compute Pool", n)
		}

		return nil
	}
}

