terraform {
  required_providers {
    confluent = {
      # source  = "confluentinc/confluent"
      # version = "2.41.0"
      source = "terraform.confluent.io/confluentinc/confluent"
    }
  }
}

provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url
    oauth_external_client_id  = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_identity_pool_id = var.oauth_identity_pool_id
    oauth_external_token_scope = "api://c8b8b903-0114-424b-8157-b832e7103367/.default"
  }
}

resource "confluent_environment" "source" {
  display_name = "Source"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_environment" "destination" {
  display_name = "Destination"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_kafka_cluster" "source" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-2"
  basic {}
  environment {
    id = confluent_environment.source.id
  }
}

resource "confluent_kafka_cluster" "destination" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-1"
  basic {}
  environment {
    id = confluent_environment.destination.id
  }
}

data "confluent_schema_registry_cluster" "source" {
  environment {
    id = confluent_environment.source.id
  }

  depends_on = [
    confluent_kafka_cluster.source
  ]
}

data "confluent_schema_registry_cluster" "destination" {
  environment {
    id = confluent_environment.destination.id
  }

  depends_on = [
    confluent_kafka_cluster.destination
  ]
}

resource "confluent_kafka_topic" "purchase_source" {
  kafka_cluster {
    id = confluent_kafka_cluster.source.id
  }
  topic_name    = "purchase_source"
  rest_endpoint = confluent_kafka_cluster.source.rest_endpoint

  depends_on = [
    confluent_kafka_cluster.source
  ]
}

resource "confluent_kafka_topic" "purchase_destination" {
  kafka_cluster {
    id = confluent_kafka_cluster.destination.id
  }
  topic_name    = "purchase_destination"
  rest_endpoint = confluent_kafka_cluster.destination.rest_endpoint

  depends_on = [
    confluent_kafka_cluster.destination
  ]
}

resource "confluent_schema" "purchase" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.purchase_source.topic_name}-value"
  format       = "PROTOBUF"
  schema       = file("./schemas/proto/purchase.proto")
}

resource "confluent_schema_exporter" "main" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.source.id
  }

  name         = "my_exporter"
  rest_endpoint = data.confluent_schema_registry_cluster.source.rest_endpoint

  context      = "schema_context"
  context_type = "CUSTOM"
  subjects     = [confluent_schema.purchase.subject_name]

  destination_schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.destination.id
    rest_endpoint = data.confluent_schema_registry_cluster.destination.rest_endpoint
  }

  reset_on_update = true
}
