terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.38.1"
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

resource "confluent_flink_statement" "old" {
  statement = <<-EOT
    INSERT INTO customers_sink (customer_id, name, address, postcode, city, email)
    SELECT customer_id, name, address, postcode, city, email
    FROM customers_source;
  EOT

  properties = {
    "sql.current-catalog"  = var.current_catalog
    "sql.current-database" = var.current_database
  }

  stopped = false
  # Step #2: Stop confluent_flink_statement.old, which will trigger the start for confluent_flink_statement.new (PENDING -> RUNNING state transition)
  # stopped = true
}

# Step #1: Create confluent_flink_statement.new, which will start from the last offsets of confluent_flink_statement.old once it's stopped
#resource "confluent_flink_statement" "new" {
#  statement = <<-EOT
#    INSERT INTO customers_sink (customer_id, name, address, postcode, city, email)
#    SELECT customer_id, name, address, postcode, city, email
#    FROM customers_source
#  EOT
#
#  properties = {
#    "sql.current-catalog"  = var.current_catalog
#    "sql.current-database" = var.current_database
#    "sql.tables.initial-offset-from" =  confluent_flink_statement.old.statement_name
#  }
#}

# Note: For more details, refer to the official Confluent documentation:
# https://docs.confluent.io/cloud/current/flink/operate-and-deploy/carry-over-offsets.html
