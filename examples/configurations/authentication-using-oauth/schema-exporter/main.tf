terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.40.0"
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

resource "confluent_schema_exporter" "order_exporter" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.main.rest_endpoint
  schema_id     = confluent_schema.order.id
  kafka_topic   = confluent_kafka_topic.order.topic_name
  # Optional: If not specified, the exporter will use the Kafka cluster associated with the Schema Registry cluster.
  kafka_cluster {
    id = confluent_kafka_cluster.standard.id
  }
}
