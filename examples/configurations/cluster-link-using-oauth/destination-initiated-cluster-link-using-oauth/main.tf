terraform {
  required_providers {
    confluent = {
      # source  = "confluentinc/confluent"
      # version = "2.35.0"
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
  }
}

data "confluent_environment" "source" {
  id = var.source_kafka_cluster_environment_id
}

data "confluent_environment" "destination" {
  id = var.destination_kafka_cluster_environment_id
}

data "confluent_kafka_cluster" "source" {
  id = var.source_kafka_cluster_id
  environment {
    id = var.source_kafka_cluster_environment_id
  }
}

data "confluent_kafka_cluster" "destination" {
  id = var.destination_kafka_cluster_id
  environment {
    id = var.destination_kafka_cluster_environment_id
  }
}

resource "confluent_cluster_link" "destination-outbound" {
  link_name = "destination-initiated-terraform-using-oauth"
  source_kafka_cluster {
    id                 = data.confluent_kafka_cluster.source.id
    bootstrap_endpoint = data.confluent_kafka_cluster.source.bootstrap_endpoint
  }

  destination_kafka_cluster {
    id            = data.confluent_kafka_cluster.destination.id
    rest_endpoint = data.confluent_kafka_cluster.destination.rest_endpoint
  }
}

resource "confluent_kafka_mirror_topic" "test" {
  source_kafka_topic {
    topic_name = var.source_topic_name
  }
  cluster_link {
    link_name = confluent_cluster_link.destination-outbound.link_name
  }
  kafka_cluster {
    id            = data.confluent_kafka_cluster.destination.id
    rest_endpoint = data.confluent_kafka_cluster.destination.rest_endpoint
  }
}
