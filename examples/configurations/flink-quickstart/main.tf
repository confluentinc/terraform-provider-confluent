terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.65.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

data "confluent_organization" "main" {}

data "confluent_environment" "staging" {
  id = var.environment_id
}

locals {
  cloud  = "AWS"
  region = "us-east-2"
  table_name = "random_int_table"
}

// In Confluent Cloud, an environment is mapped to a Flink catalog, and a Kafka cluster is mapped to a Flink database.
# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster
resource "confluent_kafka_cluster" "standard" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = local.cloud
  region       = local.region
  standard {}
  environment {
    id = data.confluent_environment.staging.id
  }
}
// Service account to perform a task within Confluent Cloud, such as executing a Flink statement
resource "confluent_service_account" "statements-runner" {
  display_name = "statements-runner"
  description  = "Service account for running Flink Statements in 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "statements-runner-environment-admin" {
  principal   = "User:${confluent_service_account.statements-runner.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = data.confluent_environment.staging.resource_name
}
// Service account that owns Flink API Key
resource "confluent_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account that has got full access to Flink resources in an environment"
}
// https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#flinkdeveloper
resource "confluent_role_binding" "app-manager-flink-developer" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "FlinkDeveloper"
  crn_pattern = data.confluent_environment.staging.resource_name
}
// https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#assigner
// https://docs.confluent.io/cloud/current/flink/operate-and-deploy/flink-rbac.html#submit-long-running-statements
resource "confluent_role_binding" "app-manager-assigner" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "Assigner"
  crn_pattern = "${data.confluent_organization.main.resource_name}/service-account=${confluent_service_account.statements-runner.id}"
}
data "confluent_flink_region" "us-east-2" {
  cloud  = local.cloud
  region = local.region
}
resource "confluent_api_key" "app-manager-flink-api-key" {
  display_name = "app-manager-flink-api-key"
  description  = "Flink API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }
  managed_resource {
    id          = data.confluent_flink_region.us-east-2.id
    api_version = data.confluent_flink_region.us-east-2.api_version
    kind        = data.confluent_flink_region.us-east-2.kind
    environment {
      id = var.environment_id
    }
  }
}
data "confluent_schema_registry_region" "essentials" {
  cloud   = local.cloud
  region  = local.region
  package = "ESSENTIALS"
}
resource "confluent_schema_registry_cluster" "essentials" {
  package = data.confluent_schema_registry_region.essentials.package
  environment {
    id = var.environment_id
  }
  region {
    # See https://docs.confluent.io/cloud/current/stream-governance/packages.html#stream-governance-regions
    id = data.confluent_schema_registry_region.essentials.id
  }
}
data "confluent_flink_region" "main" {
  cloud        = local.cloud
  region       = local.region
}

# https://docs.confluent.io/cloud/current/flink/get-started/quick-start-cloud-console.html#step-1-create-a-af-compute-pool
resource "confluent_flink_compute_pool" "main" {
  display_name = "my-compute-pool"
  cloud        = local.cloud
  region       = local.region
  max_cfu      = 10
  environment {
    id = var.environment_id
  }
  depends_on = [
    confluent_role_binding.statements-runner-environment-admin,
    confluent_role_binding.app-manager-assigner,
    confluent_role_binding.app-manager-flink-developer,
    confluent_api_key.app-manager-flink-api-key,
  ]
}
resource "confluent_flink_statement" "select-current-timestamp" {
  organization {
    id = data.confluent_organization.main.id
  }
  environment {
    id = data.confluent_environment.staging.id
  }
  compute_pool {
    id = confluent_flink_compute_pool.main.id
  }
  principal {
    id = confluent_service_account.statements-runner.id
  }
  statement     = "SELECT CURRENT_TIMESTAMP;"
  rest_endpoint = data.confluent_flink_region.main.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-flink-api-key.id
    secret = confluent_api_key.app-manager-flink-api-key.secret
  }
}
resource "confluent_flink_statement" "create-table" {
  organization {
    id = data.confluent_organization.main.id
  }
  environment {
    id = data.confluent_environment.staging.id
  }
  compute_pool {
    id = confluent_flink_compute_pool.main.id
  }
  principal {
    id = confluent_service_account.statements-runner.id
  }
  statement  = "CREATE TABLE ${local.table_name}(ts TIMESTAMP_LTZ(3), random_value INT);"
  properties = {
    "sql.current-catalog"  = data.confluent_environment.staging.display_name
    "sql.current-database" = confluent_kafka_cluster.standard.display_name
  }
  rest_endpoint = data.confluent_flink_region.main.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-flink-api-key.id
    secret = confluent_api_key.app-manager-flink-api-key.secret
  }
  depends_on = [
    confluent_flink_statement.select-current-timestamp,
  ]
}
resource "confluent_flink_statement" "insert-into-table" {
  organization {
    id = data.confluent_organization.main.id
  }
  environment {
    id = data.confluent_environment.staging.id
  }
  compute_pool {
    id = confluent_flink_compute_pool.main.id
  }
  principal {
    id = confluent_service_account.statements-runner.id
  }
  statement  = "INSERT INTO ${local.table_name} VALUES (CURRENT_TIMESTAMP, RAND_INTEGER(100)), (CURRENT_TIMESTAMP, RAND_INTEGER(1000));"
  properties = {
    "sql.current-catalog"  = data.confluent_environment.staging.display_name
    "sql.current-database" = confluent_kafka_cluster.standard.display_name
  }
  rest_endpoint = data.confluent_flink_region.main.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-flink-api-key.id
    secret = confluent_api_key.app-manager-flink-api-key.secret
  }
  depends_on = [
    confluent_flink_statement.create-table,
  ]
}
resource "confluent_flink_statement" "select-from-table" {
  organization {
    id = data.confluent_organization.main.id
  }
  environment {
    id = data.confluent_environment.staging.id
  }
  compute_pool {
    id = confluent_flink_compute_pool.main.id
  }
  principal {
    id = confluent_service_account.statements-runner.id
  }
  statement  = "SELECT * FROM ${local.table_name};"
  properties = {
    "sql.current-catalog"  = data.confluent_environment.staging.display_name
    "sql.current-database" = confluent_kafka_cluster.standard.display_name
  }
  rest_endpoint = data.confluent_flink_region.main.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-flink-api-key.id
    secret = confluent_api_key.app-manager-flink-api-key.secret
  }
  depends_on = [
    confluent_flink_statement.insert-into-table,
  ]
}
