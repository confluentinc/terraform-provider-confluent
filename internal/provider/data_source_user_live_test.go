//go:build live_test && (all || core)

// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccUserDataSourceLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Skip test if required environment variables are not set
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Use a known email for the live test - the email associated with our test account
	testUserEmail := "dayaz+tflivetesting@confluent.io"
	userDataSourceLabel := "test_live_user"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckUserDataSourceLiveConfig(endpoint, userDataSourceLabel, testUserEmail, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserDataSourceLiveExists(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel)),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel), "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel), "email", testUserEmail),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel), "full_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel), "api_version"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("data.confluent_user.%s", userDataSourceLabel), "kind"),
				),
			},
		},
	})
}

func testAccCheckUserDataSourceLiveConfig(endpoint, userDataSourceLabel, userEmail, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint          = "%s"
		cloud_api_key     = "%s"
		cloud_api_secret  = "%s"
	}

	data "confluent_user" "%s" {
		email = "%s"
	}
	`, endpoint, apiKey, apiSecret, userDataSourceLabel, userEmail)
}

func testAccCheckUserDataSourceLiveExists(resourceName string) resource.TestCheckFunc {
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