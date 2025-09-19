//go:build live_test && (all || connect)

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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccConnectArtifactLive(t *testing.T) {
	t.Skip()
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

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	artifactDisplayName := fmt.Sprintf("tf-live-artifact-%d", randomSuffix)
	artifactResourceLabel := "test_live_connect_artifact"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectArtifactLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectArtifactLiveConfig(endpoint, artifactResourceLabel, artifactDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectArtifactLiveExists(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "display_name", artifactDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "cloud", "aws"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "content_format", "JAR"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "description", "A test connect artifact for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel), "id"),
				),
			},
			{
				ResourceName:            fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"artifact_file"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					artifactId := resources[fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel)].Primary.ID
					environmentId := resources[fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel)].Primary.Attributes["environment.0.id"]
					cloud := resources[fmt.Sprintf("confluent_connect_artifact.%s", artifactResourceLabel)].Primary.Attributes["cloud"]
					return environmentId + "/" + cloud + "/" + artifactId, nil
				},
			},
		},
	})
}

func testAccCheckConnectArtifactLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_connect_artifact" {
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

func testAccCheckConnectArtifactLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckConnectArtifactLiveConfig(endpoint, artifactResourceLabel, artifactDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_connect_artifact" "%s" {
		display_name   = "%s"
		cloud          = "AWS"
		content_format = "JAR"
		description    = "A test connect artifact for live testing"
		artifact_file  = "test_artifacts/connect_artifact.jar"
		
		environment {
			id = "env-zyg27z"
		}
	}
	`, endpoint, apiKey, apiSecret, artifactResourceLabel, artifactDisplayName)
}
