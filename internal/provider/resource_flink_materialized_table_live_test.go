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

func TestAccFlinkMaterializedTableLive(t *testing.T) {
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

	// Validate required cloud credentials
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Materialized Tables depend on a pre-existing Flink compute pool, principal (service account),
	// Kafka cluster, and Flink API key. We expect these to be provisioned out-of-band and
	// supplied via environment variables, since standing them up inside this test would
	// significantly lengthen runtime and cross-cut multiple resource lifecycles.
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
			"must be set for Flink Materialized Table live tests")
	}

	// Generate a unique materialized table name to avoid collisions across parallel runs
	randomSuffix := rand.Intn(100000)
	tableDisplayName := fmt.Sprintf("tf_live_mat_table_%d", randomSuffix)
	tableResourceLabel := "test_live_flink_materialized_table"
	fullTableResourceLabel := fmt.Sprintf("confluent_flink_materialized_table.%s", tableResourceLabel)

	// The resource importer reads the IMPORT_* environment variables to reconstruct credentials
	// and parent IDs. Set them once up front (matching the wiremock test convention) and clean
	// up afterwards.
	_ = os.Setenv("IMPORT_FLINK_API_KEY", flinkApiKey)
	_ = os.Setenv("IMPORT_FLINK_API_SECRET", flinkApiSecret)
	_ = os.Setenv("IMPORT_FLINK_REST_ENDPOINT", flinkRestEndpoint)
	_ = os.Setenv("IMPORT_FLINK_PRINCIPAL_ID", flinkPrincipalId)
	_ = os.Setenv("IMPORT_CONFLUENT_ORGANIZATION_ID", organizationId)
	_ = os.Setenv("IMPORT_CONFLUENT_ENVIRONMENT_ID", environmentId)
	_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", flinkComputePoolId)
	defer func() {
		_ = os.Unsetenv("IMPORT_FLINK_API_KEY")
		_ = os.Unsetenv("IMPORT_FLINK_API_SECRET")
		_ = os.Unsetenv("IMPORT_FLINK_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_FLINK_PRINCIPAL_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ORGANIZATION_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ENVIRONMENT_ID")
		_ = os.Unsetenv("IMPORT_FLINK_COMPUTE_POOL_ID")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkMaterializedTableLiveDestroy(s, flinkRestEndpoint, organizationId, environmentId, flinkComputePoolId, flinkPrincipalId, flinkApiKey, flinkApiSecret)
		},
		Steps: []resource.TestStep{
			// Step 1: Create the materialized table with an initial query and watermark.
			{
				Config: testAccCheckFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret,
					tableResourceLabel, tableDisplayName,
					organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
					flinkRestEndpoint, flinkApiKey, flinkApiSecret, kafkaClusterId,
					"SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;",
					"col_event_time", "col_event_time - INTERVAL '5' SECOND",
					false,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkMaterializedTableLiveExists(fullTableResourceLabel),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramKafkaCluster, kafkaClusterId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramQuery, "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramWatermarkColumnName, "col_event_time"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramWatermarkExpression, "col_event_time - INTERVAL '5' SECOND"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramStopped, "false"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), organizationId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), environmentId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramRestEndpoint, flinkRestEndpoint),
					resource.TestCheckResourceAttrSet(fullTableResourceLabel, paramId),
				),
			},
			// Step 2: Update the materialized table — change the query/watermark and pause it via stopped=true.
			{
				Config: testAccCheckFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret,
					tableResourceLabel, tableDisplayName,
					organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
					flinkRestEndpoint, flinkApiKey, flinkApiSecret, kafkaClusterId,
					"SELECT user_id, product_id, price, quantity FROM orders WHERE price > 100;",
					"col_event_time", "col_event_time - INTERVAL '10' SECOND",
					true,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkMaterializedTableLiveExists(fullTableResourceLabel),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramQuery, "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 100;"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramWatermarkExpression, "col_event_time - INTERVAL '10' SECOND"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramStopped, "true"),
				),
			},
			// Step 3: Verify the resource can be re-imported cleanly using its composite ID.
			{
				ResourceName:      fullTableResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return state.RootModule().Resources[fullTableResourceLabel].Primary.ID, nil
				},
			},
		},
	})
}

func testAccCheckFlinkMaterializedTableLiveConfig(
	endpoint, apiKey, apiSecret,
	tableResourceLabel, tableDisplayName,
	organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
	flinkRestEndpoint, flinkApiKey, flinkApiSecret, kafkaClusterId,
	query, watermarkColumn, watermarkExpression string,
	stopped bool,
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
		kafka_cluster = "%s"
		query         = "%s"

		watermark_column_name = "%s"
		watermark_expression  = "%s"
		stopped               = %t
	}
	`, endpoint, apiKey, apiSecret,
		tableResourceLabel,
		organizationId, environmentId, flinkComputePoolId, flinkPrincipalId,
		flinkRestEndpoint, flinkApiKey, flinkApiSecret,
		tableDisplayName, kafkaClusterId, query,
		watermarkColumn, watermarkExpression, stopped,
	)
}

func testAccCheckFlinkMaterializedTableLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s Flink Materialized Table has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Flink Materialized Table", resourceName)
		}

		return nil
	}
}

func testAccCheckFlinkMaterializedTableLiveDestroy(s *terraform.State, flinkRestEndpoint, organizationId, environmentId, flinkComputePoolId, flinkPrincipalId, flinkApiKey, flinkApiSecret string) error {
	testClient := testAccProvider.Meta().(*Client)
	c := testClient.flinkRestClientFactory.CreateFlinkRestClient(flinkRestEndpoint, organizationId, environmentId, flinkComputePoolId, flinkPrincipalId, flinkApiKey, flinkApiSecret, false, testClient.oauthToken)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_materialized_table" {
			continue
		}
		deletedId := rs.Primary.ID
		tableName := getTableName(deletedId)
		kafkaId := getKafkaId(deletedId)
		_, response, err := executeMaterializedTableRead(c.apiContext(context.Background()), c, organizationId, environmentId, kafkaId, tableName)
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedId != "" {
			if deletedId == rs.Primary.ID {
				return fmt.Errorf("Flink Materialized Table (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
