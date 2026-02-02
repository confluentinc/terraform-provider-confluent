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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceMaterializedTableScenarioName = "confluent_flink_materialized_table Data Source Lifecycle"
)

func TestAccDataSourceMaterializedTable(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/read_materialized_table.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123/materialized-tables/table1")).
		InScenario(dataSourceMaterializedTableScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/read_materialized_table_list.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/sql/v1/organizations/1111aaaa-11aa-11aa-11aa-111111aaaaaa/environments/env-abc123/materialized-tables")).
		InScenario(dataSourceMaterializedTableScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	flinkTableDataSourceLabel := "test"
	fullTableDataSourceLabel := fmt.Sprintf("data.confluent_flink_materialized_table.%s", flinkTableDataSourceLabel)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceMaterializedTableConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMaterializedTableExists(fullTableDataSourceLabel),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramDisplayName, flinkMaterializedTableDisplayName),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramQuery, "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramWatermarkExpression, "exp123"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramWatermarkColumnName, "col123"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramStopped, "false"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramDistributedByBuckets, "10"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "distributed_by_column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullTableDataSourceLabel, "distributed_by_column_names.*", "keys"),
					resource.TestCheckTypeSetElemAttr(fullTableDataSourceLabel, "distributed_by_column_names.*", "passwords"),

					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "columns.0.columns_physical.0.column_physical_name", "user_id"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "columns.0.columns_physical.0.column_physical_kind", "Physical"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "columns.0.columns_physical.0.column_physical_comment", "string"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "columns.0.columns_physical.0.column_physical_type", "type1"),

					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "constraints.#", "1"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "constraints.0.name", "pk_orders"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "constraints.0.kind", "PRIMARY_KEY"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "constraints.0.enforced", "false"),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, "constraints.0.column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullTableDataSourceLabel, "constraints.0.column_names.*", "user_id"),
					resource.TestCheckTypeSetElemAttr(fullTableDataSourceLabel, "constraints.0.column_names.*", "product_id"),

					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullTableDataSourceLabel, paramRestEndpoint, mockServerUrl),
				),
			},
		},
	})
}

func testAccCheckDataSourceMaterializedTableConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}

	data "confluent_flink_materialized_table" "test" {
	  credentials {
		  key = "%s"
		  secret = "%s"
	  }

	  rest_endpoint = "%s"
	  principal {
		 id = "%s"
	  }
	  organization {
		 id = "%s"
	  }
	  environment {
		 id = "%s"
	  }
	  compute_pool {
		 id = "%s"
	  }
	  display_name = "%s"
	}
	`, mockServerUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkMaterializedTableDisplayName)
}
