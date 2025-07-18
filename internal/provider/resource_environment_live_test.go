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
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccEnvironmentLive(t *testing.T) {
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

	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	environmentResourceLabel := "test_live_env"
	environmentSgPackage := "ESSENTIALS"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEnvironmentLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, environmentSgPackage, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "display_name", environmentDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), getNestedStreamGovernancePackageKey(), environmentSgPackage),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_environment.%s", environmentResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update functionality
				Config: testAccCheckEnvironmentLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName+"-updated", "ADVANCED", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "display_name", environmentDisplayName+"-updated"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), getNestedStreamGovernancePackageKey(), "ADVANCED"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_environment.%s", environmentResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckEnvironmentLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, environmentSgPackage, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
		stream_governance {
			package = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, environmentSgPackage)
}
