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

func TestAccFlinkArtifactAWSLive(t *testing.T) {
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
	envDisplayName := fmt.Sprintf("tf-live-flink-artifact-env-%d", randomSuffix)
	artifactDisplayName := fmt.Sprintf("tf-live-flink-artifact-aws-%d", randomSuffix)
	envResourceLabel := "test_live_flink_artifact_env"
	artifactResourceLabel := "test_live_flink_artifact_aws"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkArtifactLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkArtifactLiveConfig(endpoint, envResourceLabel, artifactResourceLabel, envDisplayName, artifactDisplayName, "AWS", "us-east-2", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkArtifactLiveExists(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "display_name", artifactDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "region", "us-east-2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "content_format", "JAR"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "versions.#"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"artifact_file"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					artifactId := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["environment.0.id"]
					region := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["region"]
					cloud := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["cloud"]
					return fmt.Sprintf("%s/%s/%s/%s", envId, region, cloud, artifactId), nil
				},
			},
		},
	})
}

func TestAccFlinkArtifactAzureLive(t *testing.T) {
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
	envDisplayName := fmt.Sprintf("tf-live-flink-artifact-env-azure-%d", randomSuffix)
	artifactDisplayName := fmt.Sprintf("tf-live-flink-artifact-azure-%d", randomSuffix)
	envResourceLabel := "test_live_flink_artifact_env_azure"
	artifactResourceLabel := "test_live_flink_artifact_azure"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkArtifactLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkArtifactLiveConfig(endpoint, envResourceLabel, artifactResourceLabel, envDisplayName, artifactDisplayName, "AZURE", "eastus2", apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkArtifactLiveExists(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "display_name", artifactDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "cloud", "AZURE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "region", "eastus2"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "content_format", "JAR"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"artifact_file"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					artifactId := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["environment.0.id"]
					region := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["region"]
					cloud := resources[fmt.Sprintf("confluent_flink_artifact.%s", artifactResourceLabel)].Primary.Attributes["cloud"]
					return fmt.Sprintf("%s/%s/%s/%s", envId, region, cloud, artifactId), nil
				},
			},
		},
	})
}

func testAccCheckFlinkArtifactLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Flink Artifact is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_artifact" {
			continue
		}
		deletedArtifactId := rs.Primary.ID
		environmentId := rs.Primary.Attributes["environment.0.id"]
		cloud := rs.Primary.Attributes["cloud"]
		region := rs.Primary.Attributes["region"]
		req := c.faClient.FlinkArtifactsArtifactV1Api.GetArtifactV1FlinkArtifact(c.faApiContext(context.Background()), deletedArtifactId).Region(region).Cloud(cloud).Environment(environmentId)
		deletedArtifact, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedArtifact.Id != nil {
			if *deletedArtifact.Id == rs.Primary.ID {
				return fmt.Errorf("Flink Artifact (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckFlinkArtifactLiveConfig(endpoint, envResourceLabel, artifactResourceLabel, envDisplayName, artifactDisplayName, cloud, region, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
	}

	resource "confluent_flink_artifact" "%s" {
		display_name = "%s"
		cloud = "%s"
		region = "%s"
		content_format = "JAR"
		artifact_file = "test_artifacts/connect_artifact.jar"
		runtime_language = "JAVA"
		description = "Test Flink Artifact for live testing"
		documentation_link = "https://github.com/confluentinc/terraform-provider-confluent"
		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, envResourceLabel, envDisplayName, artifactResourceLabel, artifactDisplayName, cloud, region, envResourceLabel)
}

func testAccCheckFlinkArtifactLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Flink Artifact has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Flink Artifact", n)
		}

		return nil
	}
}

