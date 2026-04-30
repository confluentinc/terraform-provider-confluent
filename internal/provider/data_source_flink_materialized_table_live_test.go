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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceFlinkMaterializedTableLive(t *testing.T) {
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
		endpoint = "https://api.confluent.cloud"
	}

	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	organizationId := os.Getenv("ORGANIZATION_ID")
	environmentId := os.Getenv("ENVIRONMENT_ID")
	flinkComputePoolId := os.Getenv("FLINK_COMPUTE_POOL_ID")
	flinkPrincipalId := os.Getenv("FLINK_PRINCIPAL_ID")
	flinkRestEndpoint := os.Getenv("FLINK_REST_ENDPOINT")
	flinkApiKey := os.Getenv("FLINK_API_KEY")
	flinkApiSecret := os.Getenv("FLINK_API_SECRET")
	kafkaClusterId := os.Getenv("KAFKA_CLUSTER_ID")

	if organizationId == "" || environmentId == "" || flinkComputePoolId == "" ||
		flinkPrincipalId == "" || flinkRestEndpoint == "" || flinkApiKey == "" ||
		flinkApiSecret == "" || kafkaClusterId == "" {
		t.Fatal("ORGANIZATION_ID, ENVIRONMENT_ID, FLINK_COMPUTE_POOL_ID, FLINK_PRINCIPAL_ID, " +
			"FLINK_REST_ENDPOINT, FLINK_API_KEY, FLINK_API_SECRET, and KAFKA_CLUSTER_ID " +
			"must be set for Flink Materialized Table data source live tests")
	}

	randomSuffix := rand.Intn(100000)
	tableDisplayName := fmt.Sprintf("tf_live_mat_table_ds_%d", randomSuffix)
	tableResourceLabel := "test_live_flink_materialized_table_ds"
	tableDataSourceLabel := "test_live_flink_materialized_table_ds_lookup"
	fullTableDataSourceLabel := fmt.Sprintf("data.confluent_flink_materialized_table.%s", tableDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// The companion resource is destroyed by the resource_flink_materialized_table_live_test
		// destroy helper; reuse it here so we don't leak the table created during this test.
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkMaterializedTableLiveDestroy(s, flinkRestEndpoint, organizationId, environmentId, flinkComputePoolId, flinkPrincipalId, flinkApiKey, flinkApiSecret)
		},
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret,
					tableResourceLabel, tableDataSourceLabel, tableDisplayName,
					organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
					flinkRestEndpoint, flinkApiKey, flinkApiSecret, kafkaClusterId,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceFlinkMaterializedTableLiveExists(fullTableDataSourceLabel),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClusterId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), organizationId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), environmentId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramRestEndpoint, flinkRestEndpoint),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, paramId),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, paramQuery),
				),
			},
		},
	})
}

func testAccCheckDataSourceFlinkMaterializedTableLiveConfig(
	endpoint, apiKey, apiSecret,
	tableResourceLabel, tableDataSourceLabel, tableDisplayName,
	organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
	flinkRestEndpoint, flinkApiKey, flinkApiSecret, kafkaClusterId string,
) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_flink_materialized_table" "%s" {
		organization {
			id = "%s"
		}
		environment {
			id = "%s"
		}
		compute_pool {
			id = "%s"
		}
		principal {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}

		display_name  = "%s"
		kafka_cluster {
			id = "%s"
		}
		query         = "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;"
	}

	data "confluent_flink_materialized_table" "%s" {
		organization {
			id = "%s"
		}
		environment {
			id = "%s"
		}
		compute_pool {
			id = "%s"
		}
		principal {
			id = "%s"
		}
		rest_endpoint = "%s"
		credentials {
			key    = "%s"
			secret = "%s"
		}

		display_name = confluent_flink_materialized_table.%s.display_name
	}
	`, endpoint, apiKey, apiSecret,
		tableResourceLabel,
		organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
		flinkRestEndpoint, flinkApiKey, flinkApiSecret,
		tableDisplayName, kafkaClusterId,
		tableDataSourceLabel,
		organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
		flinkRestEndpoint, flinkApiKey, flinkApiSecret,
		tableResourceLabel,
	)
}

func testAccCheckDataSourceFlinkMaterializedTableLiveExists(dataSourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("%s Flink Materialized Table data source has not been found", dataSourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Flink Materialized Table data source", dataSourceName)
		}

		return nil
	}
}
