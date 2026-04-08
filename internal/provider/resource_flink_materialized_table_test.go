package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

const (
	scenarioStateMaterializedTableHasBeenCreated = "A new materialized table has been just created"
	scenarioStateMaterializedTableHasBeenUpdated = "A materialized table has been updated"
	scenarioStateMaterializedTableHasBeenDeleted = "The materialized table has been deleted"
	materializedTableScenarioName                = "confluent_flink_materialized_table Resource Lifecycle"

	flinkMaterializedTableDisplayName = "table1"
	flinkMaterializedTableDatabase    = "lkc01"
)

var createFlinkMaterializedTablePath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/databases/%s/materialized-tables", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkMaterializedTableDatabase)
var readFlinkMaterializedTablePath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/databases/%s/materialized-tables/%s", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkMaterializedTableDatabase, flinkMaterializedTableDisplayName)

func TestAccFlinkMaterializedTable(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockTestServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createFlinkMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/create_materialized_table.json")
	createFlinkMaterializedTableStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateMaterializedTableHasBeenCreated).
		WillReturn(
			string(createFlinkMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkMaterializedTableStub)

	readCreatedMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/read_materialized_table.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenCreated).
		WillReturn(
			string(readCreatedMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateFlinkMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/update_materialized_table.json")
	updateFlinkConnectionStub := wiremock.Put(wiremock.URLPathEqualTo(readFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenCreated).
		WillSetStateTo(scenarioStateMaterializedTableHasBeenUpdated).
		WillReturn(
			string(updateFlinkMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(updateFlinkConnectionStub)

	readUpdatedFlinkMaterializedTableResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/read_materialized_table_updated.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenUpdated).
		WillReturn(
			string(readUpdatedFlinkMaterializedTableResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteConnectionStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenUpdated).
		WillSetStateTo(scenarioStateMaterializedTableHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteConnectionStub)

	readDeletedConnectionResponse, _ := ioutil.ReadFile("../testdata/flink_materialized_table/read_deleted_materialized_table.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkMaterializedTablePath)).
		InScenario(materializedTableScenarioName).
		WhenScenarioStateIs(scenarioStateMaterializedTableHasBeenDeleted).
		WillReturn(
			string(readDeletedConnectionResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	flinkMaterializedTableResourceLabel := "test"
	fullMaterializedTableResourceLabel := fmt.Sprintf("confluent_flink_materialized_table.%s", flinkMaterializedTableResourceLabel)
	//distributedByColumns := []string{"keys", "passwords"}

	_ = os.Setenv("API_KEY", flinkAPIKeyUpdated)
	_ = os.Setenv("IMPORT_FLINK_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_FLINK_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_FLINK_REST_ENDPOINT", mockTestServerUrl)
	_ = os.Setenv("IMPORT_FLINK_PRINCIPAL_ID", flinkPrincipalIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ORGANIZATION_ID", flinkOrganizationIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ENVIRONMENT_ID", flinkEnvironmentIdTest)
	_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", flinkComputePoolIdTest)
	defer func() {
		_ = os.Unsetenv("API_KEY")
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
			return testAccCheckMaterializedTableDestroy(s, mockTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckMaterializedTableConfig(mockTestServerUrl, flinkMaterializedTableResourceLabel, flinkAPIKey),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMaterializedTableExists(fullMaterializedTableResourceLabel),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramDisplayName, flinkMaterializedTableDisplayName),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramQuery, "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramWatermarkExpression, "exp123"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramWatermarkColumnName, "col123"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramStopped, "false"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramDistributedByBuckets, "10"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.*", "keys"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.*", "passwords"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_name", "user_id"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_kind", "Physical"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_comment", "string"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_type", "type1"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.#", "1"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.name", "pk_orders"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.kind", "PRIMARY_KEY"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.enforced", "false"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.*", "user_id"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.*", "product_id"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramRestEndpoint, mockTestServerUrl),
				),
			},
			{
				Config: testAccCheckMaterializedTableConfigUpdated(mockTestServerUrl, flinkMaterializedTableResourceLabel, flinkAPIKeyUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckMaterializedTableExists(fullMaterializedTableResourceLabel),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramDisplayName, flinkMaterializedTableDisplayName),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramQuery, "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 100;"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramWatermarkExpression, "exp1234"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramWatermarkColumnName, "col1234"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramStopped, "true"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramDistributedByBuckets, "10"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.*", "keys"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "distributed_by_column_names.*", "passwords"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.#", "3"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_name", "user_id"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_kind", "Physical"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_comment", "string"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.0.columns_physical.0.column_physical_type", "type1"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.1.columns_physical.0.column_physical_name", "user_id2"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.1.columns_physical.0.column_physical_kind", "Physical"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.1.columns_physical.0.column_physical_comment", "string2"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.1.columns_physical.0.column_physical_type", "type2"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_name", "user_id3"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_kind", "Computed"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_comment", "string3"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_type", "type3"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_expression", "expression1"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "columns.2.columns_computed.0.column_computed_virtual", "true"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.#", "2"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.name", "pk_orders"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.kind", "PRIMARY_KEY"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.enforced", "false"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.*", "user_id"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.0.column_names.*", "product_id"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.1.name", "pk_orders2"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.1.kind", "PRIMARY_KEY"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.1.enforced", "true"),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, "constraints.1.column_names.#", "2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.1.column_names.*", "user_id2"),
					resource.TestCheckTypeSetElemAttr(fullMaterializedTableResourceLabel, "constraints.1.column_names.*", "product_id2"),

					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolUpdatedIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalUpdatedIdTest),
					resource.TestCheckResourceAttr(fullMaterializedTableResourceLabel, paramRestEndpoint, mockTestServerUrl),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullMaterializedTableResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fullMaterializedTableResourceLabel].Primary.ID
					return resourceId, nil
				},
			},
		},
	})
}

func testAccCheckMaterializedTableConfig(mockServerUrl, resourceLabel, apikey string) string {
	return fmt.Sprintf(`
	provider "confluent" {
    	endpoint = "%s"
	}

	resource "confluent_flink_materialized_table" "%s" {
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
      display_name  = "%s"
	  kafka_cluster = "%s"
      stopped = "false"
	  query = "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 1000;"
      watermark_column_name = "col123"
	  watermark_expression = "exp123"
	  distributed_by_buckets = 10
      distributed_by_column_names = [
    	"keys",
    	"passwords"
      ]
	constraints {
      name = "pk_orders"
      kind = "PRIMARY_KEY"
      column_names = ["user_id","product_id"]
      enforced = false
      }
	columns {
		columns_physical {
			column_physical_name = "user_id"
	        column_physical_kind = "Physical"
	  		column_physical_comment = "string"
			column_physical_type = "type1"
		}
	}
}
	`, mockServerUrl, resourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkMaterializedTableDisplayName, flinkMaterializedTableDatabase)
}

func testAccCheckMaterializedTableConfigUpdated(mockServerUrl, resourceLabel, apikey string) string {
	return fmt.Sprintf(`
	provider "confluent" {
    	endpoint = "%s"
	}

	resource "confluent_flink_materialized_table" "%s" {
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
      display_name  = "%s"
	  kafka_cluster = "%s"
      stopped = "true"
	  query = "SELECT user_id, product_id, price, quantity FROM orders WHERE price > 100;"
      watermark_column_name = "col1234"
	  watermark_expression = "exp1234"
	  distributed_by_buckets = 10
      distributed_by_column_names = [
    	"keys",
    	"passwords"
      ]
	constraints {
      name = "pk_orders"
      kind = "PRIMARY_KEY"
      column_names = ["user_id","product_id"]
      enforced = false
      }
	constraints {
      name = "pk_orders2"
      kind = "PRIMARY_KEY"
      column_names = ["user_id2","product_id2"]
      enforced = true
      }
	columns {
		columns_physical {
			column_physical_name = "user_id"
	        column_physical_kind = "Physical"
	  		column_physical_comment = "string"
			column_physical_type = "type1"
		}
	}
	columns {
		columns_physical {
			column_physical_name = "user_id2"
	        column_physical_kind = "Physical"
	  		column_physical_comment = "string2"
			column_physical_type = "type2"
		}
	}
	columns {
		columns_computed {
			column_computed_name = "user_id3"
	        column_computed_kind = "Computed"
	  		column_computed_comment = "string3"
			column_computed_type = "type3"
			column_computed_expression = "expression1"
			column_computed_virtual = true
		}
	}
}
	`, mockServerUrl, resourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalUpdatedIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolUpdatedIdTest, flinkMaterializedTableDisplayName, flinkMaterializedTableDatabase)
}

func testAccCheckMaterializedTableExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s materialized table has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s materialized table", n)
		}

		return nil
	}
}
func testAccCheckMaterializedTableDestroy(s *terraform.State, url string) error {
	testClient := testAccProvider.Meta().(*Client)
	c := testClient.flinkRestClientFactory.CreateFlinkRestClient(url, flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkPrincipalIdTest, kafkaApiKey, kafkaApiSecret, false, testClient.oauthToken)
	// Loop through the resources in state, verifying each Kafka topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_materialized_table" {
			continue
		}
		deletedId := rs.Primary.ID
		_, response, err := c.apiClient.ConnectionsSqlV1Api.GetSqlv1Connection(c.apiContext(context.Background()), flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkConnectionNameTest).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedId != "" {
			// Otherwise return the error
			if deletedId == rs.Primary.ID {
				return fmt.Errorf("materialized table (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
