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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccKafkaClustersDataSourceLive(t *testing.T) {
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

	kafkaClustersDataSourceLabel := "test_live_kafka_clusters_data_source"
	fullKafkaClustersDataSourceLabel := fmt.Sprintf("data.confluent_kafka_clusters.%s", kafkaClustersDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClustersDataSourceLabel, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.id", "lkc-7g3pzj"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.api_version", "cmk/v2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.kind", "Cluster"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.display_name", "(DO NOT DELETE) Standard AWS Kafka cluster used by TF Live Tests"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.availability", "HIGH"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.bootstrap_endpoint", "SASL_SSL://pkc-921jm.us-east-2.aws.confluent.cloud:9092"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.cloud", "AWS"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.basic.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.standard.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.standard.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.enterprise.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.freight.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.dedicated.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.byok_key.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.byok_key.0.id", ""),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.endpoints.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.network.0.id", ""),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.rest_endpoint", "https://pkc-921jm.us-east-2.aws.confluent.cloud:443"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.0.rbac_crn"),

					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.id", "lkc-gr63rv"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.api_version", "cmk/v2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.kind", "Cluster"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.display_name", "(DO NOT DELETE) Dedicated AWS Kafka cluster used by Live Tests"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.availability", "MULTI_ZONE"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.bootstrap_endpoint", "SASL_SSL://pkc-v1j3qj.us-east-2.aws.confluent.cloud:9092"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.cloud", "AWS"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.region", "us-east-2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.basic.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.enterprise.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.freight.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.0.%", "3"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.0.cku", "2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.0.encryption_key", ""),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.0.zones.#", "3"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.byok_key.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.byok_key.0.id", ""),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.endpoints.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.environment.0.id", "env-zyg27z"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.network.0.id", ""),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.rest_endpoint", "https://pkc-v1j3qj.us-east-2.aws.confluent.cloud:443"),
					resource.TestCheckResourceAttrSet(fullKafkaClustersDataSourceLabel, "clusters.1.rbac_crn"),
				),
			},
		},
	})
}

func testAccCheckKafkaClustersDataSourceLiveConfig(endpoint, kafkaClustersDataSourceLabel, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	data "confluent_kafka_clusters" "%s" {
		environment {
			id = "env-zyg27z"
		}
	}
	`, endpoint, apiKey, apiSecret, kafkaClustersDataSourceLabel)
}
