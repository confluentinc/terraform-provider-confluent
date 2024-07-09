terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.80.0"
    }
  }
}

provider "confluent" {
  organization_id       = var.organization_id            # optionally use CONFLUENT_ORGANIZATION_ID env var
  environment_id        = var.environment_id             # optionally use CONFLUENT_ENVIRONMENT_ID env var
  flink_compute_pool_id = var.flink_compute_pool_id      # optionally use FLINK_COMPUTE_POOL_ID env var
  flink_rest_endpoint   = var.flink_rest_endpoint        # optionally use FLINK_REST_ENDPOINT env var
  flink_api_key         = var.flink_api_key              # optionally use FLINK_API_KEY env var
  flink_api_secret      = var.flink_api_secret           # optionally use FLINK_API_SECRET env var
  flink_principal_id    = var.flink_principal_id         # optionally use FLINK_PRINCIPAL_ID env var
}
locals {
  table_name = "random_int_table"
}

resource "confluent_flink_statement" "select-current-timestamp" {
  statement     = "SELECT CURRENT_TIMESTAMP;"
}
resource "confluent_flink_statement" "create-table" {
  statement  = "CREATE TABLE ${local.table_name}(ts TIMESTAMP_LTZ(3), random_value INT);"
  properties = {
    "sql.current-catalog"  = var.current_catalog
    "sql.current-database" = var.current_database
  }
  depends_on = [
    confluent_flink_statement.select-current-timestamp,
  ]
}
resource "confluent_flink_statement" "insert-into-table" {
  statement  = "INSERT INTO ${local.table_name} VALUES (CURRENT_TIMESTAMP, RAND_INTEGER(100)), (CURRENT_TIMESTAMP, RAND_INTEGER(1000));"
  properties = {
    "sql.current-catalog"  = var.current_catalog
    "sql.current-database" = var.current_database
  }
  depends_on = [
    confluent_flink_statement.create-table,
  ]
}
resource "confluent_flink_statement" "select-from-table" {
  statement  = "SELECT * FROM ${local.table_name};"
  properties = {
    "sql.current-catalog"  = var.current_catalog
    "sql.current-database" = var.current_database
  }
  depends_on = [
    confluent_flink_statement.insert-into-table,
  ]
}
