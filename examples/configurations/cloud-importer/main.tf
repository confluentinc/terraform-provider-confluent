terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.46.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_tf_importer" "example" {}
