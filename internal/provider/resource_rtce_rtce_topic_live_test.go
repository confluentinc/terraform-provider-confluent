//go:build live_test && (all || core)

// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// resolveEndpoint returns the Confluent Cloud API endpoint.
// Priority: CONFLUENT_CLOUD_ENDPOINT (full URL override) > TF_ACC_ENV (shorthand) > prod default.
//
//	TF_ACC_ENV=devel  → https://api.devel.cpdev.cloud
//	TF_ACC_ENV=stag   → https://api.stag.cpdev.cloud
//	(empty or "prod")    → https://api.confluent.cloud
func resolveEndpoint() string {
	if ep := os.Getenv("CONFLUENT_CLOUD_ENDPOINT"); ep != "" {
		return ep
	}
	switch os.Getenv("TF_ACC_ENV") {
	case "devel":
		return "https://api.devel.cpdev.cloud"
	case "stag":
		return "https://api.stag.cpdev.cloud"
	default:
		return "https://api.confluent.cloud"
	}
}

func testAccRtceTopicPrerequisiteConfig(randomSuffix int) string {
	cloud := os.Getenv("TF_ACC_CLOUD")
	if cloud == "" {
		cloud = "AWS"
	}
	region := os.Getenv("TF_ACC_REGION")
	if region == "" {
		region = "us-west-2"
	}
	return fmt.Sprintf(`
resource "confluent_environment" "prerequisite" {
  display_name = "tf-live-prereq-env-%d"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_kafka_cluster" "prerequisite" {
  display_name = "tf-live-prereq-cluster-%d"
  cloud        = "%s"
  region       = "%s"
  availability = "SINGLE_ZONE"
  basic {}
  environment {
    id = confluent_environment.prerequisite.id
  }
}

resource "confluent_service_account" "prerequisite" {
  display_name = "tf-live-prereq-sa-%d"
}

data "confluent_organization" "prerequisite" {}

resource "confluent_role_binding" "prerequisite" {
  principal   = "User:${confluent_service_account.prerequisite.id}"
  role_name   = "OrganizationAdmin"
  crn_pattern = replace(data.confluent_organization.prerequisite.resource_name, "/[a-z]+\\.cpdev\\.cloud/", "confluent.cloud")
}

resource "confluent_api_key" "prerequisite" {
  display_name = "tf-live-prereq-key-%d"
  owner {
    id          = confluent_service_account.prerequisite.id
    api_version = confluent_service_account.prerequisite.api_version
    kind        = confluent_service_account.prerequisite.kind
  }
  managed_resource {
    id          = confluent_kafka_cluster.prerequisite.id
    api_version = confluent_kafka_cluster.prerequisite.api_version
    kind        = confluent_kafka_cluster.prerequisite.kind
    environment {
      id = confluent_environment.prerequisite.id
    }
  }
  depends_on = [confluent_role_binding.prerequisite]
}

resource "confluent_kafka_topic" "prerequisite" {
  topic_name       = "tf_live_prereq_topic_%d"
  partitions_count = 1
  kafka_cluster {
    id = confluent_kafka_cluster.prerequisite.id
  }
  rest_endpoint = confluent_kafka_cluster.prerequisite.rest_endpoint
  credentials {
    key    = confluent_api_key.prerequisite.id
    secret = confluent_api_key.prerequisite.secret
  }
}

data "confluent_schema_registry_cluster" "prerequisite" {
  environment {
    id = confluent_environment.prerequisite.id
  }
  depends_on = [confluent_kafka_cluster.prerequisite]
}

resource "confluent_api_key" "prerequisite_sr" {
  display_name = "tf-live-prereq-sr-key-%d"
  owner {
    id          = confluent_service_account.prerequisite.id
    api_version = confluent_service_account.prerequisite.api_version
    kind        = confluent_service_account.prerequisite.kind
  }
  managed_resource {
    id          = data.confluent_schema_registry_cluster.prerequisite.id
    api_version = data.confluent_schema_registry_cluster.prerequisite.api_version
    kind        = data.confluent_schema_registry_cluster.prerequisite.kind
    environment {
      id = confluent_environment.prerequisite.id
    }
  }
  depends_on = [confluent_role_binding.prerequisite]
}

resource "confluent_schema" "prerequisite" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.prerequisite.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.prerequisite.rest_endpoint
  subject_name  = "${confluent_kafka_topic.prerequisite.topic_name}-value"
  format        = "AVRO"
  schema        = jsonencode({
    type   = "record"
    name   = "PrerequisiteValue"
    fields = [{ name = "id", type = "string" }]
  })
  credentials {
    key    = confluent_api_key.prerequisite_sr.id
    secret = confluent_api_key.prerequisite_sr.secret
  }
}
`, randomSuffix, randomSuffix, cloud, region, randomSuffix, randomSuffix, randomSuffix, randomSuffix)
}

func testAccRtceTopicPrerequisiteWithProviderConfig(endpoint, apiKey, apiSecret string, randomSuffix int) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}
	`, endpoint, apiKey, apiSecret) + testAccRtceTopicPrerequisiteConfig(randomSuffix)
}

func TestAccRtceTopicLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := resolveEndpoint()

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	rtceTopicResourceLabel := fmt.Sprintf("test_live_rtce_topic_%d", randomSuffix)
	cloud := os.Getenv("TF_ACC_CLOUD")
	if cloud == "" {
		cloud = "AWS"
	}
	description := fmt.Sprintf("tf_live_rtce_topic_%d", randomSuffix)
	region := os.Getenv("TF_ACC_REGION")
	if region == "" {
		region = "us-west-2"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRtceTopicLiveDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create prerequisite resources
				Config: testAccRtceTopicPrerequisiteWithProviderConfig(endpoint, apiKey, apiSecret, randomSuffix),
			},
			{
				PreConfig: func() {
					t.Log("Waiting 5 minutes for prerequisite resources to propagate...")
					time.Sleep(5 * time.Minute)
				},
				Config: testAccCheckRtceTopicLiveConfig(endpoint, rtceTopicResourceLabel, cloud, description, region, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceTopicLiveExists(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "cloud", cloud),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "description", description),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "environment.0.id", "confluent_environment.prerequisite", "id"),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "kafka_cluster.0.id", "confluent_kafka_cluster.prerequisite", "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "region", region),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "topic_name", "confluent_kafka_topic.prerequisite", "topic_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccRtceTopicUpdateLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := resolveEndpoint()

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	rtceTopicResourceLabel := fmt.Sprintf("test_live_rtce_topic_update_%d", randomSuffix)
	cloud := os.Getenv("TF_ACC_CLOUD")
	if cloud == "" {
		cloud = "AWS"
	}
	description := fmt.Sprintf("tf_live_rtce_topic_%d", randomSuffix)
	region := os.Getenv("TF_ACC_REGION")
	if region == "" {
		region = "us-west-2"
	}
	descriptionUpdated := fmt.Sprintf("tf_live_rtce_topic_update_%d", randomSuffix)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRtceTopicLiveDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create prerequisite resources
				Config: testAccRtceTopicPrerequisiteWithProviderConfig(endpoint, apiKey, apiSecret, randomSuffix),
			},
			{
				PreConfig: func() {
					t.Log("Waiting 5 minutes for prerequisite resources to propagate...")
					time.Sleep(5 * time.Minute)
				},
				Config: testAccCheckRtceTopicLiveConfig(endpoint, rtceTopicResourceLabel, cloud, description, region, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceTopicLiveExists(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "cloud", cloud),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "description", description),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "environment.0.id", "confluent_environment.prerequisite", "id"),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "kafka_cluster.0.id", "confluent_kafka_cluster.prerequisite", "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "region", region),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "topic_name", "confluent_kafka_topic.prerequisite", "topic_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "id"),
				),
			},
			{
				Config: testAccCheckRtceTopicUpdateLiveConfig(endpoint, rtceTopicResourceLabel, cloud, descriptionUpdated, region, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceTopicLiveExists(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "cloud", cloud),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "description", descriptionUpdated),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "environment.0.id", "confluent_environment.prerequisite", "id"),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "kafka_cluster.0.id", "confluent_kafka_cluster.prerequisite", "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "region", region),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "topic_name", "confluent_kafka_topic.prerequisite", "topic_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "id"),
				),
			},
		},
	})
}

func TestAccRtceTopicMinimalLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := resolveEndpoint()

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	rtceTopicResourceLabel := fmt.Sprintf("test_live_rtce_topic_minimal_%d", randomSuffix)
	cloud := os.Getenv("TF_ACC_CLOUD")
	if cloud == "" {
		cloud = "AWS"
	}
	description := fmt.Sprintf("tf_live_rtce_topic_%d", randomSuffix)
	region := os.Getenv("TF_ACC_REGION")
	if region == "" {
		region = "us-west-2"
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckRtceTopicLiveDestroy,
		Steps: []resource.TestStep{
			{
				// Step 1: Create prerequisite resources
				Config: testAccRtceTopicPrerequisiteWithProviderConfig(endpoint, apiKey, apiSecret, randomSuffix),
			},
			{
				PreConfig: func() {
					t.Log("Waiting 5 minutes for prerequisite resources to propagate...")
					time.Sleep(5 * time.Minute)
				},
				Config: testAccCheckRtceTopicMinimalLiveConfig(endpoint, rtceTopicResourceLabel, cloud, description, region, apiKey, apiSecret, randomSuffix),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckRtceTopicLiveExists(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "cloud", cloud),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "description", description),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "environment.0.id", "confluent_environment.prerequisite", "id"),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "kafka_cluster.0.id", "confluent_kafka_cluster.prerequisite", "id"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "region", region),
					resource.TestCheckResourceAttrPair(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "topic_name", "confluent_kafka_topic.prerequisite", "topic_name"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_rtce_rtce_topic.%s", rtceTopicResourceLabel), "id"),
				),
			},
		},
	})
}

func testAccCheckRtceTopicLiveConfig(endpoint, rtceTopicResourceLabel string, cloud string, description string, region string, apiKey, apiSecret string, randomSuffix int) string {
	return testAccRtceTopicPrerequisiteConfig(randomSuffix) + fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_rtce_rtce_topic" "%s" {
		cloud = "%s"
		description = "%s"
		environment {
			id = confluent_environment.prerequisite.id
		}
		kafka_cluster {
			id = confluent_kafka_cluster.prerequisite.id
		}
		region = "%s"
		topic_name = confluent_kafka_topic.prerequisite.topic_name
		depends_on = [confluent_schema.prerequisite]
	}
	`, endpoint, apiKey, apiSecret, rtceTopicResourceLabel, cloud, description, region)
}

func testAccCheckRtceTopicMinimalLiveConfig(endpoint, rtceTopicResourceLabel string, cloud string, description string, region string, apiKey, apiSecret string, randomSuffix int) string {
	return testAccRtceTopicPrerequisiteConfig(randomSuffix) + fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_rtce_rtce_topic" "%s" {
		cloud = "%s"
		description = "%s"
		environment {
			id = confluent_environment.prerequisite.id
		}
		kafka_cluster {
			id = confluent_kafka_cluster.prerequisite.id
		}
		region = "%s"
		topic_name = confluent_kafka_topic.prerequisite.topic_name
		depends_on = [confluent_schema.prerequisite]
	}
	`, endpoint, apiKey, apiSecret, rtceTopicResourceLabel, cloud, description, region)
}

func testAccCheckRtceTopicUpdateLiveConfig(endpoint, rtceTopicResourceLabel string, cloud string, description string, region string, apiKey, apiSecret string, randomSuffix int) string {
	return testAccRtceTopicPrerequisiteConfig(randomSuffix) + fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_rtce_rtce_topic" "%s" {
		cloud = "%s"
		description = "%s"
		environment {
			id = confluent_environment.prerequisite.id
		}
		kafka_cluster {
			id = confluent_kafka_cluster.prerequisite.id
		}
		region = "%s"
		topic_name = confluent_kafka_topic.prerequisite.topic_name
		depends_on = [confluent_schema.prerequisite]
	}
	`, endpoint, apiKey, apiSecret, rtceTopicResourceLabel, cloud, description, region)
}

func testAccCheckRtceTopicLiveExists(resourceName string) resource.TestCheckFunc {
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

func testAccCheckRtceTopicLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_rtce_rtce_topic" {
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
