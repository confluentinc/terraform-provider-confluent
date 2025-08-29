terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.38.1"
    }
  }
}

locals {
  cloud  = "AWS"
  region = "us-east-2"
}

provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url
    oauth_external_client_id  = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_identity_pool_id = var.oauth_identity_pool_id
  }
}

data "confluent_organization" "main" {}

resource "confluent_environment" "staging-oauth" {
  display_name = "Staging_OAuth"

  stream_governance {
    package = "ESSENTIALS"
  }
}

# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster
resource "confluent_kafka_cluster" "standard" {
  display_name = "standard_cluster"
  availability = "SINGLE_ZONE"
  cloud        = local.cloud
  region       = local.region
  standard {}
  environment {
    id = confluent_environment.staging-oauth.id
  }
}

data "confluent_schema_registry_cluster" "main" {
  environment {
    id = confluent_environment.staging-oauth.id
  }

  depends_on = [
    confluent_kafka_cluster.standard
  ]
}

resource "confluent_kafka_topic" "order" {
  kafka_cluster {
    id = confluent_kafka_cluster.standard.id
  }
  topic_name    = "order"
  rest_endpoint = confluent_kafka_cluster.standard.rest_endpoint
}

data "confluent_kafka_topic" "order_data" {
  kafka_cluster {
    id = confluent_kafka_cluster.standard.id
  }
  topic_name    = confluent_kafka_topic.order.topic_name
  rest_endpoint = confluent_kafka_cluster.standard.rest_endpoint
}

resource "confluent_schema" "order" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.main.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.order.topic_name}-value"
  format       = "AVRO"
  schema       = file("./schemas/avro/order.avsc")
}

data "confluent_flink_region" "us-east-2" {
  cloud  = local.cloud
  region = local.region
}

resource "confluent_flink_compute_pool" "main" {
  display_name = "my-compute-pool"
  cloud        = local.cloud
  region       = local.region
  max_cfu      = 10
  environment {
    id = confluent_environment.staging-oauth.id
  }
}

resource "confluent_flink_statement" "read-orders-source-table" {
  organization {
    id = data.confluent_organization.main.id
  }
  environment {
    id = confluent_environment.staging-oauth.id
  }
  compute_pool {
    id = confluent_flink_compute_pool.main.id
  }
  principal {
    id = var.oauth_identity_pool_id
  }
  # https://docs.confluent.io/cloud/current/flink/reference/example-data.html#marketplace-database
  statement = file("./statements/query-orders-source-table.sql")
  properties = {
    "sql.current-catalog"  = confluent_environment.staging-oauth.display_name
    "sql.current-database" = confluent_kafka_cluster.standard.display_name
  }
  rest_endpoint = data.confluent_flink_region.us-east-2.rest_endpoint

  depends_on = [
    confluent_schema.order
  ]
}
