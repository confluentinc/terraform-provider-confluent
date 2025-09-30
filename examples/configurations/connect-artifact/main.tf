terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.43.0"
    }
  }
}

provider "confluent" {
  cloud_api_key         = var.confluent_cloud_api_key
  cloud_api_secret      = var.confluent_cloud_api_secret
}

data "confluent_environment" "example" {
  id = var.environment_id
}

resource "confluent_connect_artifact" "main" {
  display_name = var.artifact_display_name
  cloud        = var.artifact_cloud
  environment {
    id = var.environment_id
  }
  content_format = var.artifact_content_format
  artifact_file  = var.artifact_file
  description    = var.artifact_description
}

# Create a fully-managed connector that uses the custom SMT
resource "confluent_connector" "datagen_with_custom_smt" {
  depends_on = [confluent_connect_artifact.main]

  environment {
    id = var.environment_id
  }
  kafka_cluster {
    id = var.kafka_cluster_id
  }
  config_nonsensitive = {
    "connector.class" = "DatagenSource"
    "name"           = "datagen-custom-smt"
    "tasks.max"      = "1"
    "kafka.topic"    = "datagen-source-custom-smt"
    "max.interval"   = "3000"
    "output.data.format" = "JSON"
    "quickstart"     = "ORDERS"
    "transforms"     = "customSMT"
    "transforms.customSMT.type" = "io.confluent.connect.transforms.ExtractTopic$Value"
    "transforms.customSMT.custom.smt.artifact.id" = confluent_connect_artifact.main.id
  }
  config_sensitive = {
    "transforms.customSMT.field" = "itemid"
    "kafka.api.key" = var.kafka_api_key
    "kafka.api.secret" = var.kafka_api_secret
  }
}