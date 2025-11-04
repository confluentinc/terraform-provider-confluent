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
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccProviderIntegrationV1AWSLive(t *testing.T) {
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

	// AWS IAM role ARN for provider integration
	awsIamRoleArn := os.Getenv("TEST_KMS_KEY_ID")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	if awsIamRoleArn == "" {
		t.Skip("Skipping Provider Integration AWS test. TEST_AWS_IAM_ROLE_ARN must be set to run this test.")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	envDisplayName := fmt.Sprintf("tf-live-pi-env-%d", randomSuffix)
	piDisplayName := fmt.Sprintf("tf-live-provider-integration-aws-%d", randomSuffix)
	envResourceLabel := "test_live_pi_env"
	piResourceLabel := "test_live_provider_integration_aws"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckProviderIntegrationLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckProviderIntegrationAWSLiveConfig(endpoint, envResourceLabel, piResourceLabel, envDisplayName, piDisplayName, awsIamRoleArn, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderIntegrationLiveExists(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel), "display_name", piDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel), "aws.0.customer_role_arn", awsIamRoleArn),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel), "id"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel), "aws.0.iam_role_arn"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel), "aws.0.external_id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					piId := resources[fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel)].Primary.ID
					envId := resources[fmt.Sprintf("confluent_provider_integration.%s", piResourceLabel)].Primary.Attributes["environment.0.id"]
					return fmt.Sprintf("%s/%s", envId, piId), nil
				},
			},
		},
	})
}

func testAccCheckProviderIntegrationLiveDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each Provider Integration is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_provider_integration" {
			continue
		}
		deletedPIId := rs.Primary.ID
		environmentId := rs.Primary.Attributes["environment.0.id"]
		req := c.piClient.IntegrationsPimV1Api.GetPimV1Integration(c.piApiContext(context.Background()), deletedPIId).Environment(environmentId)
		deletedPI, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedPI.Id != nil {
			if *deletedPI.Id == rs.Primary.ID {
				return fmt.Errorf("Provider Integration (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckProviderIntegrationAWSLiveConfig(endpoint, envResourceLabel, piResourceLabel, envDisplayName, piDisplayName, awsIamRoleArn, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		cloud_api_key = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
	}

	resource "confluent_provider_integration" "%s" {
		display_name = "%s"
		environment {
			id = confluent_environment.%s.id
		}
		aws {
			customer_role_arn = "%s"
		}
	}
	`, endpoint, apiKey, apiSecret, envResourceLabel, envDisplayName, piResourceLabel, piDisplayName, envResourceLabel, awsIamRoleArn)
}

func testAccCheckProviderIntegrationLiveExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Provider Integration has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Provider Integration", n)
		}

		return nil
	}
}

