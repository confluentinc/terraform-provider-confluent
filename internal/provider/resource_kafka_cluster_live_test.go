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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccKafkaClusterLive(t *testing.T) {
	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	clusterDisplayName := fmt.Sprintf("tf-provider-live-kafka-test-%d", time.Now().Unix())
	environmentDisplayName := fmt.Sprintf("tf-provider-live-env-test-%d", time.Now().Unix())
	clusterResourceLabel := "test_live_kafka_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "SINGLE_ZONE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					clusterId := resources[fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)].Primary.ID
					environmentId := resources[fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + clusterId, nil
				},
			},
		},
	})
}

func testAccCheckKafkaClusterLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_environment" "%s" {
		display_name = "%s"
		stream_governance {
			package = "ESSENTIALS"
		}
	}

	resource "confluent_kafka_cluster" "%s" {
		display_name = "%s"
		availability = "SINGLE_ZONE"
		cloud        = "AWS"
		region       = "us-east-1"
		standard {}

		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, environmentResourceLabel)
}
