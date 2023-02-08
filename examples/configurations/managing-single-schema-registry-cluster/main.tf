terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.29.0"
    }
  }
}

provider "confluent" {
  schema_registry_id            = var.schema_registry_id
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint
  schema_registry_api_key       = var.schema_registry_api_key
  schema_registry_api_secret    = var.schema_registry_api_secret
}

resource "confluent_schema" "purchase-v1" {
  subject_name = "purchase-value"
  format = "PROTOBUF"
  schema = file("./schemas/proto/purchase.proto")
}
