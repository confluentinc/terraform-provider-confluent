terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.65.0"
    }
  }
}

provider "confluent" {
  kafka_id            = var.kafka_id
  kafka_rest_endpoint = var.kafka_rest_endpoint
  kafka_api_key       = var.kafka_api_key
  kafka_api_secret    = var.kafka_api_secret
}

resource "confluent_kafka_topic" "main" {
  for_each = var.topics

  topic_name       = each.key
  partitions_count = each.value.partitions_count
  config           = each.value.config

  lifecycle {
    prevent_destroy = true
  }
}
