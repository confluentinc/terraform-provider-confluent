//go:build live_test && (all || kafka)

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

// Test Basic cluster - simplest, fastest test
func TestAccKafkaClusterBasicLive(t *testing.T) {
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

	clusterDisplayName := fmt.Sprintf("tf-live-basic-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_basic_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterBasicLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "SINGLE_ZONE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "basic.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rest_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
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

// Test Standard cluster - production-ready with extended feature set
func TestAccKafkaClusterStandardLive(t *testing.T) {
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

	clusterDisplayName := fmt.Sprintf("tf-live-standard-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_standard_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterStandardLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "SINGLE_ZONE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "standard.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rest_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
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

// Test Enterprise cluster - high-performance, multi-zone
func TestAccKafkaClusterEnterpriseLive(t *testing.T) {
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

	clusterDisplayName := fmt.Sprintf("tf-live-enterprise-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_enterprise_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterEnterpriseLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "HIGH"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "enterprise.#", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rest_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
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

// Test Dedicated cluster with private networking - requires network dependency
func TestAccKafkaClusterDedicatedWithNetworkLive(t *testing.T) {
	// Disable dedicated tests until cost is figured out
	t.Skip()
	// Enable parallel execution for I/O bound operations (Dedicated takes ~45 minutes)
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

	clusterDisplayName := fmt.Sprintf("tf-live-dedicated-net-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	networkDisplayName := fmt.Sprintf("tf-live-network-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_dedicated_network_cluster"
	environmentResourceLabel := "test_live_env"
	networkResourceLabel := "test_live_network"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterDedicatedWithNetworkLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "SINGLE_ZONE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.#", "1"),
					// Dedicated cluster specific attributes per API docs
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.0.cku", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.0.zones.#"),
					// Dedicated clusters should have endpoints
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rest_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "network.0.id"),
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

// Test Dedicated cluster - CKU-based with optional networking and encryption
func TestAccKafkaClusterDedicatedLive(t *testing.T) {
	// Disable dedicated tests until cost is figured out
	t.Skip()
	// Enable parallel execution for I/O bound operations (Dedicated takes ~45 minutes)
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

	clusterDisplayName := fmt.Sprintf("tf-live-dedicated-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_dedicated_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterDedicatedLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "SINGLE_ZONE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.#", "1"),
					// Dedicated cluster specific attributes per API docs
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.0.cku", "1"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "dedicated.0.zones.#"),
					// Dedicated clusters should have endpoints
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "bootstrap_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rest_endpoint"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
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

// Test Freight cluster - high-scale with relaxed latency requirements
func TestAccKafkaClusterFreightLive(t *testing.T) {
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

	clusterDisplayName := fmt.Sprintf("tf-live-freight-%d", rand.Intn(1000000))
	environmentDisplayName := fmt.Sprintf("tf-live-env-%d", rand.Intn(1000000))
	clusterResourceLabel := "test_live_freight_cluster"
	environmentResourceLabel := "test_live_env"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClusterFreightLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "display_name", clusterDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "availability", "HIGH"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "region", "us-east-1"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "freight.#", "1"),
					// Freight clusters should have zones information
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "freight.0.zones.#"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_kafka_cluster.%s", clusterResourceLabel), "rbac_crn"),
					// Note: bootstrap_endpoint and rest_endpoint are optional for Freight clusters per API docs
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

// Configuration for Basic cluster
func testAccCheckKafkaClusterBasicLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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
		basic {}

		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, environmentResourceLabel)
}

// Configuration for Standard cluster
func testAccCheckKafkaClusterStandardLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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

// Configuration for Enterprise cluster
func testAccCheckKafkaClusterEnterpriseLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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
		availability = "HIGH"
		cloud        = "AWS"
		region       = "us-east-1"
		enterprise {}

		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, environmentResourceLabel)
}

// Configuration for Dedicated cluster with private networking
func testAccCheckKafkaClusterDedicatedWithNetworkLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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

	resource "confluent_network" "%s" {
		display_name     = "%s"
		cloud            = "AWS"
		region           = "us-east-1"
		connection_types = ["PRIVATELINK"]
		
		environment {
			id = confluent_environment.%s.id
		}
	}

	resource "confluent_kafka_cluster" "%s" {
		display_name = "%s"
		availability = "SINGLE_ZONE"
		cloud        = "AWS"
		region       = "us-east-1"
		dedicated {
			cku = 1
		}

		environment {
			id = confluent_environment.%s.id
		}

		network {
			id = confluent_network.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, networkResourceLabel, networkDisplayName, environmentResourceLabel, clusterResourceLabel, clusterDisplayName, environmentResourceLabel, networkResourceLabel)
}

// Configuration for Dedicated cluster
func testAccCheckKafkaClusterDedicatedLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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
		dedicated {
			cku = 1
		}

		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, environmentResourceLabel)
}

// Configuration for Freight cluster
func testAccCheckKafkaClusterFreightLiveConfig(endpoint, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, apiKey, apiSecret string) string {
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
		availability = "HIGH"
		cloud        = "AWS"
		region       = "us-east-1"
		freight {}

		environment {
			id = confluent_environment.%s.id
		}
	}
	`, endpoint, apiKey, apiSecret, environmentResourceLabel, environmentDisplayName, clusterResourceLabel, clusterDisplayName, environmentResourceLabel)
}
