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
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Shared infra the live tests target (also hardcoded in the existing live
// tests for ksql, role_binding, etc.).
const (
	flinkMaterializedTableLiveOrganizationId = "424fb7bf-40c2-433f-81a5-c45942a6a539"
	flinkMaterializedTableLiveEnvironmentId  = "env-zyg27z"
)

// The Flink compute pool created by the test must live in the same AWS region
// as the Kafka cluster it references. We don't have a Vault-supplied region,
// so derive it from KAFKA_STANDARD_AWS_REST_ENDPOINT (e.g. ".us-east-1.aws.").
var awsRegionInKafkaEndpoint = regexp.MustCompile(`\.([a-z]+-[a-z]+-\d+)\.aws\.`)

func extractAwsRegionFromKafkaRestEndpoint(endpoint string) (string, error) {
	m := awsRegionInKafkaEndpoint.FindStringSubmatch(endpoint)
	if len(m) < 2 {
		return "", fmt.Errorf("could not parse AWS region from %q", endpoint)
	}
	return m[1], nil
}

func TestAccFlinkMaterializedTableLive(t *testing.T) {
	t.Parallel()

	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud"
	}
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	kafkaClusterId := os.Getenv("KAFKA_STANDARD_AWS_CLUSTER_ID")
	kafkaRestEndpoint := os.Getenv("KAFKA_STANDARD_AWS_REST_ENDPOINT")
	if kafkaClusterId == "" || kafkaRestEndpoint == "" {
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID and KAFKA_STANDARD_AWS_REST_ENDPOINT must be set for Flink Materialized Table live tests")
	}
	region, err := extractAwsRegionFromKafkaRestEndpoint(kafkaRestEndpoint)
	if err != nil {
		t.Fatalf("could not derive AWS region for Flink compute pool: %s", err)
	}

	randomSuffix := rand.Intn(100000)
	tableDisplayName := fmt.Sprintf("tf_live_mat_table_%d", randomSuffix)
	tableResourceLabel := "test_live_flink_materialized_table"
	saResourceLabel := "live_flink_mt_sa"
	poolResourceLabel := "live_flink_mt_pool"
	apiKeyResourceLabel := "live_flink_mt_api_key"
	regionDataSourceLabel := "live_flink_mt_region"
	fullTableResourceLabel := fmt.Sprintf("confluent_flink_materialized_table.%s", tableResourceLabel)

	// Prevent IMPORT_* env vars set in ImportStateIdFunc from leaking to other parallel tests.
	t.Cleanup(unsetFlinkMaterializedTableImportEnv)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkMaterializedTableLiveDestroy,
		Steps: []resource.TestStep{
			// Query examples.marketplace.orders, a built-in read-only catalog
			// every Confluent Cloud Flink workspace exposes, so the test does
			// not need to seed source data.
			{
				Config: testAccCheckFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret, region, kafkaClusterId,
					saResourceLabel, poolResourceLabel, apiKeyResourceLabel, regionDataSourceLabel,
					tableResourceLabel, tableDisplayName, randomSuffix,
					"SELECT order_id, customer_id, product_id, price FROM examples.marketplace.orders WHERE price > 100",
					false,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkMaterializedTableLiveExists(fullTableResourceLabel),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClusterId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramStopped, "false"),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), flinkMaterializedTableLiveOrganizationId),
					resource.TestCheckResourceAttr(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkMaterializedTableLiveEnvironmentId),
					resource.TestCheckResourceAttrSet(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId)),
					resource.TestCheckResourceAttrSet(fullTableResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId)),
					resource.TestCheckResourceAttrSet(fullTableResourceLabel, paramRestEndpoint),
					resource.TestCheckResourceAttrSet(fullTableResourceLabel, paramId),
				),
			},
			{
				Config: testAccCheckFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret, region, kafkaClusterId,
					saResourceLabel, poolResourceLabel, apiKeyResourceLabel, regionDataSourceLabel,
					tableResourceLabel, tableDisplayName, randomSuffix,
					"SELECT order_id, customer_id, product_id, price FROM examples.marketplace.orders WHERE price > 100",
					true,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkMaterializedTableLiveExists(fullTableResourceLabel),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableResourceLabel, paramStopped, "true"),
				),
			},
			{
				ResourceName:      fullTableResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				// Import re-derives credentials and rest_endpoint from IMPORT_* env vars rather than
				// the original config block, so they may not byte-match what's in state.
				ImportStateVerifyIgnore: []string{"credentials.0.key", "credentials.0.secret", "rest_endpoint"},
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs := state.RootModule().Resources[fullTableResourceLabel]
					keyRs := state.RootModule().Resources[fmt.Sprintf("confluent_api_key.%s", apiKeyResourceLabel)]
					saRs := state.RootModule().Resources[fmt.Sprintf("confluent_service_account.%s", saResourceLabel)]
					poolRs := state.RootModule().Resources[fmt.Sprintf("confluent_flink_compute_pool.%s", poolResourceLabel)]
					regionRs := state.RootModule().Resources[fmt.Sprintf("data.confluent_flink_region.%s", regionDataSourceLabel)]
					if rs == nil || keyRs == nil || saRs == nil || poolRs == nil || regionRs == nil {
						return "", fmt.Errorf("could not locate prerequisite resources in state for import step")
					}
					// ImportStateIdFunc runs immediately before the import, so env vars set here are
					// visible to the importer (which reads them via extractFlink*).
					_ = os.Setenv("IMPORT_FLINK_API_KEY", keyRs.Primary.ID)
					_ = os.Setenv("IMPORT_FLINK_API_SECRET", keyRs.Primary.Attributes["secret"])
					_ = os.Setenv("IMPORT_FLINK_REST_ENDPOINT", regionRs.Primary.Attributes[paramRestEndpoint])
					_ = os.Setenv("IMPORT_FLINK_PRINCIPAL_ID", saRs.Primary.ID)
					_ = os.Setenv("IMPORT_CONFLUENT_ORGANIZATION_ID", flinkMaterializedTableLiveOrganizationId)
					_ = os.Setenv("IMPORT_CONFLUENT_ENVIRONMENT_ID", flinkMaterializedTableLiveEnvironmentId)
					_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", poolRs.Primary.ID)
					return rs.Primary.ID, nil
				},
			},
		},
	})
}

func unsetFlinkMaterializedTableImportEnv() {
	for _, k := range []string{
		"IMPORT_FLINK_API_KEY",
		"IMPORT_FLINK_API_SECRET",
		"IMPORT_FLINK_REST_ENDPOINT",
		"IMPORT_FLINK_PRINCIPAL_ID",
		"IMPORT_CONFLUENT_ORGANIZATION_ID",
		"IMPORT_CONFLUENT_ENVIRONMENT_ID",
		"IMPORT_FLINK_COMPUTE_POOL_ID",
	} {
		_ = os.Unsetenv(k)
	}
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

func testAccCheckFlinkMaterializedTableLiveDestroy(_ *terraform.State) error {
	return nil
}

func testAccCheckFlinkMaterializedTableLiveConfig(
	endpoint, apiKey, apiSecret, region, kafkaClusterId,
	saResourceLabel, poolResourceLabel, apiKeyResourceLabel, regionDataSourceLabel,
	tableResourceLabel, tableDisplayName string,
	randomSuffix int,
	query string,
	stopped bool,
) string {
	return fmt.Sprintf(`
provider "confluent" {
    endpoint         = "%s"
    cloud_api_key    = "%s"
    cloud_api_secret = "%s"
}

data "confluent_flink_region" "%s" {
    cloud  = "AWS"
    region = "%s"
}

resource "confluent_service_account" "%s" {
    display_name = "tf-live-flink-mt-sa-%d"
    description  = "Service account for Flink Materialized Table live test"
}

resource "confluent_role_binding" "%s_developer" {
    principal   = "User:${confluent_service_account.%s.id}"
    role_name   = "FlinkDeveloper"
    crn_pattern = "crn://confluent.cloud/organization=%s/environment=%s"
}

resource "confluent_role_binding" "%s_env_admin" {
    principal   = "User:${confluent_service_account.%s.id}"
    role_name   = "EnvironmentAdmin"
    crn_pattern = "crn://confluent.cloud/organization=%s/environment=%s"
}

resource "confluent_role_binding" "%s_assigner" {
    principal   = "User:${confluent_service_account.%s.id}"
    role_name   = "Assigner"
    crn_pattern = "crn://confluent.cloud/organization=%s/service-account=${confluent_service_account.%s.id}"
}

resource "confluent_flink_compute_pool" "%s" {
    display_name = "tf-live-flink-mt-pool-%d"
    cloud        = "AWS"
    region       = "%s"
    max_cfu      = 5
    environment {
        id = "%s"
    }
}

resource "confluent_api_key" "%s" {
    display_name = "tf-live-flink-mt-key-%d"
    description  = "Flink API Key for Materialized Table live test"

    owner {
        id          = confluent_service_account.%s.id
        api_version = confluent_service_account.%s.api_version
        kind        = confluent_service_account.%s.kind
    }

    managed_resource {
        id          = data.confluent_flink_region.%s.id
        api_version = data.confluent_flink_region.%s.api_version
        kind        = data.confluent_flink_region.%s.kind
        environment {
            id = "%s"
        }
    }

    depends_on = [
        confluent_role_binding.%s_developer,
        confluent_role_binding.%s_env_admin,
        confluent_role_binding.%s_assigner,
    ]
}

resource "confluent_flink_materialized_table" "%s" {
    organization {
        id = "%s"
    }
    environment {
        id = "%s"
    }
    compute_pool {
        id = confluent_flink_compute_pool.%s.id
    }
    principal {
        id = confluent_service_account.%s.id
    }
    rest_endpoint = data.confluent_flink_region.%s.rest_endpoint
    credentials {
        key    = confluent_api_key.%s.id
        secret = confluent_api_key.%s.secret
    }

    display_name = "%s"
    kafka_cluster {
        id = "%s"
    }
    query   = "%s"
    stopped = %t
}
`,
		endpoint, apiKey, apiSecret,
		regionDataSourceLabel, region,
		saResourceLabel, randomSuffix,
		saResourceLabel, saResourceLabel, flinkMaterializedTableLiveOrganizationId, flinkMaterializedTableLiveEnvironmentId,
		saResourceLabel, saResourceLabel, flinkMaterializedTableLiveOrganizationId, flinkMaterializedTableLiveEnvironmentId,
		saResourceLabel, saResourceLabel, flinkMaterializedTableLiveOrganizationId, saResourceLabel,
		poolResourceLabel, randomSuffix, region, flinkMaterializedTableLiveEnvironmentId,
		apiKeyResourceLabel, randomSuffix,
		saResourceLabel, saResourceLabel, saResourceLabel,
		regionDataSourceLabel, regionDataSourceLabel, regionDataSourceLabel,
		flinkMaterializedTableLiveEnvironmentId,
		saResourceLabel, saResourceLabel, saResourceLabel,
		tableResourceLabel,
		flinkMaterializedTableLiveOrganizationId,
		flinkMaterializedTableLiveEnvironmentId,
		poolResourceLabel,
		saResourceLabel,
		regionDataSourceLabel,
		apiKeyResourceLabel, apiKeyResourceLabel,
		tableDisplayName, kafkaClusterId,
		query, stopped,
	)
}
