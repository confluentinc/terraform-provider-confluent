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
		t.Fatal("KAFKA_STANDARD_AWS_CLUSTER_ID and KAFKA_STANDARD_AWS_REST_ENDPOINT must be set for Flink Materialized Table data source live tests")
	}
	region, err := extractAwsRegionFromKafkaRestEndpoint(kafkaRestEndpoint)
	if err != nil {
		t.Fatalf("could not derive AWS region for Flink compute pool: %s", err)
	}

	randomSuffix := rand.Intn(100000)
	tableDisplayName := fmt.Sprintf("tf_live_mat_table_ds_%d", randomSuffix)
	// Distinct labels from the resource live test so both can run in parallel
	// without colliding on Terraform state names.
	tableResourceLabel := "test_live_flink_materialized_table_ds"
	tableDataSourceLabel := "test_live_flink_materialized_table_ds_lookup"
	saResourceLabel := "live_flink_mt_ds_sa"
	poolResourceLabel := "live_flink_mt_ds_pool"
	apiKeyResourceLabel := "live_flink_mt_ds_api_key"
	regionDataSourceLabel := "live_flink_mt_ds_region"
	fullTableDataSourceLabel := fmt.Sprintf("data.confluent_flink_materialized_table.%s", tableDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckFlinkMaterializedTableLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceFlinkMaterializedTableLiveConfig(
					endpoint, apiKey, apiSecret, region, kafkaClusterId,
					saResourceLabel, poolResourceLabel, apiKeyResourceLabel, regionDataSourceLabel,
					tableResourceLabel, tableDataSourceLabel, tableDisplayName, randomSuffix,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckDataSourceFlinkMaterializedTableLiveExists(fullTableDataSourceLabel),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramDisplayName, tableDisplayName),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramKafkaCluster, paramId), kafkaClusterId),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), "424fb7bf-40c2-433f-81a5-c45942a6a539"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), "env-zyg27z"),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId)),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId)),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, paramRestEndpoint),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, paramId),
					resource.TestCheckResourceAttrSet(fullTableDataSourceLabel, paramQuery),
				),
			},
		},
	})
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

func testAccCheckDataSourceFlinkMaterializedTableLiveConfig(
	endpoint, apiKey, apiSecret, region, kafkaClusterId,
	saResourceLabel, poolResourceLabel, apiKeyResourceLabel, regionDataSourceLabel,
	tableResourceLabel, tableDataSourceLabel, tableDisplayName string,
	randomSuffix int,
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
    display_name = "tf-live-flink-mt-ds-sa-%d"
    description  = "Service account for Flink Materialized Table data source live test"
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
    display_name = "tf-live-flink-mt-ds-pool-%d"
    cloud        = "AWS"
    region       = "%s"
    max_cfu      = 5
    environment {
        id = "%s"
    }
}

resource "confluent_api_key" "%s" {
    display_name = "tf-live-flink-mt-ds-key-%d"
    description  = "Flink API Key for Materialized Table data source live test"

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
    query = "SELECT order_id, customer_id, product_id, price FROM examples.marketplace.orders WHERE price > 100"
}

data "confluent_flink_materialized_table" "%s" {
    organization {
        id = "%s"
    }
    environment {
        id = "%s"
    }
    compute_pool {
        id = confluent_flink_compute_pool.%s.id
    }
    rest_endpoint = data.confluent_flink_region.%s.rest_endpoint
    credentials {
        key    = confluent_api_key.%s.id
        secret = confluent_api_key.%s.secret
    }

    display_name = confluent_flink_materialized_table.%s.display_name
}
`,
		endpoint, apiKey, apiSecret,
		regionDataSourceLabel, region,
		saResourceLabel, randomSuffix,
		saResourceLabel, saResourceLabel, "424fb7bf-40c2-433f-81a5-c45942a6a539", "env-zyg27z",
		saResourceLabel, saResourceLabel, "424fb7bf-40c2-433f-81a5-c45942a6a539", "env-zyg27z",
		saResourceLabel, saResourceLabel, "424fb7bf-40c2-433f-81a5-c45942a6a539", saResourceLabel,
		poolResourceLabel, randomSuffix, region, "env-zyg27z",
		apiKeyResourceLabel, randomSuffix,
		saResourceLabel, saResourceLabel, saResourceLabel,
		regionDataSourceLabel, regionDataSourceLabel, regionDataSourceLabel,
		"env-zyg27z",
		saResourceLabel, saResourceLabel, saResourceLabel,
		tableResourceLabel,
		"424fb7bf-40c2-433f-81a5-c45942a6a539",
		"env-zyg27z",
		poolResourceLabel,
		saResourceLabel,
		regionDataSourceLabel,
		apiKeyResourceLabel, apiKeyResourceLabel,
		tableDisplayName, kafkaClusterId,
		tableDataSourceLabel,
		"424fb7bf-40c2-433f-81a5-c45942a6a539",
		"env-zyg27z",
		poolResourceLabel,
		regionDataSourceLabel,
		apiKeyResourceLabel, apiKeyResourceLabel,
		tableResourceLabel,
	)
}
