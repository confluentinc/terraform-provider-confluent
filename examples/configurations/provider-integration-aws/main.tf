terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.42.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "staging" {
  display_name = "Staging"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_provider_integration" "main" {
  environment {
    id = confluent_environment.staging.id
  }
  aws {
    customer_role_arn = var.customer_role_arn
  }
  display_name = "provider_integration_main"
}
