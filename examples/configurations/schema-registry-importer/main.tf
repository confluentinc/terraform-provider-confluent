terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.50.0"
    }
  }
}

provider "confluent" {
  schema_registry_id = var.schema_registry_id
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint
  schema_registry_api_key = var.schema_registry_api_key
  schema_registry_api_secret = var.schema_registry_api_secret
}

resource "confluent_tf_importer" "example" {}
