terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.37.0"
    }
  }
}

provider "confluent" {
  schema_registry_id            = var.schema_registry_id
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint
  schema_registry_api_key       = var.schema_registry_api_key
  schema_registry_api_secret    = var.schema_registry_api_secret
}

resource "confluent_tag" "pii" {
  name        = "PII"
  description = "PII tag description"
}

resource "confluent_schema_registry_kek" "aws_kek" {
  name        = "kek-name"
  kms_type    = "aws-kms"
  kms_key_id  = var.aws_kms_key_arn
  shared      = false
  hard_delete = true
}

resource "confluent_schema" "purchase" {
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "purchase-value"
  format       = "AVRO"
  # tag is also explicitly defined in purchase.avsc ("confluent:tags": ["PII"]) "for customer_id" field
  schema       = file("./schemas/avro/purchase.avsc")

  ruleset {
    domain_rules {
      name   = "encryptPII"
      kind   = "TRANSFORM"
      type   = "ENCRYPT"
      mode   = "WRITEREAD"
      tags   = [confluent_tag.pii.name]
      params = {
        "encrypt.kek.name" = confluent_schema_registry_kek.aws_kek.name
      }
      on_failure = "ERROR,NONE"
    }
  }
  hard_delete = true
}
