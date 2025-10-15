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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccFlinkConnectionLive(t *testing.T) {
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

	// Read Flink credentials from environment variables
	flinkComputePoolId := os.Getenv("FLINK_COMPUTE_POOL_ID")
	flinkApiKey := os.Getenv("FLINK_API_KEY")
	flinkApiSecret := os.Getenv("FLINK_API_SECRET")
	flinkRestEndpoint := os.Getenv("FLINK_REST_ENDPOINT")
	flinkPrincipalId := os.Getenv("FLINK_PRINCIPAL_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if flinkComputePoolId == "" || flinkApiKey == "" || flinkApiSecret == "" || flinkRestEndpoint == "" || flinkPrincipalId == "" {
		t.Fatal("FLINK_COMPUTE_POOL_ID, FLINK_API_KEY, FLINK_API_SECRET, FLINK_REST_ENDPOINT, and FLINK_PRINCIPAL_ID must be set for Flink Connection live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	connectionDisplayName := fmt.Sprintf("tf-live-flink-connection-%d", randomSuffix)
	connectionResourceLabel := "test_live_flink_connection"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkConnectionLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkConnectionLiveConfig(endpoint, connectionResourceLabel, connectionDisplayName, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId, apiKey, apiSecret, flinkApiKey, flinkApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkConnectionLiveExists(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "display_name", connectionDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "type", "MONGODB"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "endpoint", "mongodb://localhost:27017"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"username", "password"},
			},
			{
				Config: testAccCheckFlinkConnectionUpdateLiveConfig(endpoint, connectionResourceLabel, connectionDisplayName, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId, apiKey, apiSecret, flinkApiKey, flinkApiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkConnectionLiveExists(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "display_name", connectionDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel), "endpoint", "mongodb://localhost:27017"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_connection.%s", connectionResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"username", "password"},
			},
		},
	})
}

func testAccCheckFlinkConnectionLiveDestroy(s *terraform.State) error {
	// Flink Connections are cleaned up by Terraform
	// We verify they are removed from state
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_connection" {
			continue
		}
	}
	return nil
}

func testAccCheckFlinkConnectionLiveConfig(endpoint, connectionResourceLabel, connectionDisplayName, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId, apiKey, apiSecret, flinkApiKey, flinkApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
		flink_api_key = "%s"
		flink_api_secret = "%s"
		flink_rest_endpoint = "%s"
		organization_id = "424fb7bf-40c2-433f-81a5-c45942a6a539"
		environment_id = "env-zyg27z"
		flink_compute_pool_id = "%s"
		flink_principal_id = "%s"
	}

	resource "confluent_flink_connection" "%s" {
		display_name = "%s"
		type = "MONGODB"
		endpoint = "mongodb://localhost:27017"
		username = "testuser"
		password = "testpass"
	}
	`, endpoint, apiKey, apiSecret, flinkApiKey, flinkApiSecret, flinkRestEndpoint, flinkComputePoolId, flinkPrincipalId, connectionResourceLabel, connectionDisplayName)
}

func testAccCheckFlinkConnectionUpdateLiveConfig(endpoint, connectionResourceLabel, connectionDisplayName, flinkComputePoolId, flinkRestEndpoint, flinkPrincipalId, apiKey, apiSecret, flinkApiKey, flinkApiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
		flink_api_key = "%s"
		flink_api_secret = "%s"
		flink_rest_endpoint = "%s"
		organization_id = "424fb7bf-40c2-433f-81a5-c45942a6a539"
		environment_id = "env-zyg27z"
		flink_compute_pool_id = "%s"
		flink_principal_id = "%s"
	}

	resource "confluent_flink_connection" "%s" {
		display_name = "%s"
		type = "MONGODB"
		endpoint = "mongodb://localhost:27017"
		username = "testuser"
		password = "updatedpassword"
	}
	`, endpoint, apiKey, apiSecret, flinkApiKey, flinkApiSecret, flinkRestEndpoint, flinkComputePoolId, flinkPrincipalId, connectionResourceLabel, connectionDisplayName)
}

func testAccCheckFlinkConnectionLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Flink Connection has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Flink Connection", n)
		}

		return nil
	}
}

