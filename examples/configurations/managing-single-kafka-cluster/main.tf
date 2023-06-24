terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.46.0"
    }
  }
}

provider "confluent" {
  cloud_api_key       = var.confluent_cloud_api_key
  cloud_api_secret    = var.confluent_cloud_api_secret

  kafka_id            = var.kafka_id
  kafka_rest_endpoint = var.kafka_rest_endpoint
  kafka_api_key       = var.kafka_api_key
  kafka_api_secret    = var.kafka_api_secret
}

resource "confluent_kafka_topic" "orders" {
  topic_name    = "orders"
}
resource "confluent_kafka_acl" "app-producer-write-on-topic" {
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${data.confluent_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
}

data "confluent_service_account" "app-producer" {
  display_name = "app-producer"
}
