//go:build live_test && (all || [GROUP_NAME] || drift)

// Copyright 2025 Confluent Inc. All Rights Reserved.
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
)

// TestAcc[ResourceName]DriftDetection validates the provider's ability to accurately
// read and represent [resource_name] resources from Confluent Cloud.
//
// Test approach:
// - Uses a persistent, manually created resource
// - Reads via data source (no resource destruction)
// - Validates attributes match expected values
//
// Test behavior:
// - PASS: Provider correctly reads all attributes; no drift detected
// - FAIL: Attributes don't match (drift detected)
func TestAcc[ResourceName]DriftDetection(t *testing.T) {
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping drift detection test. Set TF_ACC_PROD=1 to run this test.")
	}

	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}

	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for drift detection tests")
	}

	// ========================================
	// CONFIGURATION: Update these to match your manually created resource
	// ========================================
	resourceId := "[RESOURCE_ID]"                    // e.g., "env-xxxxx", "pool-xxxxx"
	// parentId := "[PARENT_ID]"                     // Uncomment if resource has parent
	
	expectedAttribute1 := "[EXPECTED_VALUE_1]"
	expectedAttribute2 := "[EXPECTED_VALUE_2]"
	// Add more expected attributes as needed
	// ========================================

	dataSourceLabel := "drift_test_[resource_name]"
	dataSourceName := fmt.Sprintf("data.confluent_[resource_name].%s", dataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheck[ResourceName]DriftConfig(endpoint, apiKey, apiSecret, dataSourceLabel, resourceId),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", resourceId),
					resource.TestCheckResourceAttr(dataSourceName, "attribute1", expectedAttribute1),
					resource.TestCheckResourceAttr(dataSourceName, "attribute2", expectedAttribute2),
					// Add more checks as needed
				),
			},
		},
	})
}

// testAccCheck[ResourceName]DriftConfig generates Terraform config using a data source.
// Data sources are read-only and never trigger resource destruction.
func testAccCheck[ResourceName]DriftConfig(endpoint, apiKey, apiSecret, dataSourceLabel, resourceId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	data "confluent_[resource_name]" "%s" {
		id = "%s"
		# Add parent reference if needed:
		# parent_resource {
		#   id = "[PARENT_ID]"
		# }
	}
	`, endpoint, apiKey, apiSecret, dataSourceLabel, resourceId)
}

/*
USAGE:
1. Copy to: internal/provider/resource_[RESOURCE_NAME]_drift_test.go
2. Replace [PLACEHOLDER] values (ResourceName, resource_name, GROUP_NAME, etc.)
3. Update resource ID and expected attributes in CONFIGURATION section
4. Customize data source config to match your resource schema
5. Manually create the resource in Confluent Cloud
6. Run: make live-test-drift

EXAMPLES:
- Identity Pool: resource_identity_pool_drift_test.go
- Identity Provider: resource_identity_provider_drift_test.go
*/
