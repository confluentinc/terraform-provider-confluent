terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.62.0"
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

resource "confluent_gateway" "main" {
  display_name = "aws-ingress-private-link-gateway"
  environment {
    id = confluent_environment.staging.id
  }
  aws_ingress_private_link_gateway {
    region = var.region
  }
}

resource "confluent_access_point" "main" {
  display_name = "aws-ingress-private-link-access-point"
  environment {
    id = confluent_environment.staging.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_ingress_private_link_endpoint {
    vpc_endpoint_id = var.vpc_endpoint_id
  }
  depends_on = [
    confluent_gateway.main
  ]
}
