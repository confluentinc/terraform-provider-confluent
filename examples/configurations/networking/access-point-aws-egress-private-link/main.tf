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

data "confluent_environment" "main" {
  id = var.environment_id
}

resource "confluent_gateway" "main" {
  display_name = "aws-egress-private-link-gateway"
  environment {
    id = data.confluent_environment.main.id
  }
  aws_egress_private_link_gateway {
    region = var.region
  }
}

resource "confluent_access_point" "main" {
  display_name = "aws-egress-private-link-access-point"
  environment {
    id = data.confluent_environment.main.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_egress_private_link_endpoint {
    vpc_endpoint_service_name = var.vpc_endpoint_service_name
    enable_high_availability  = var.enable_high_availability
  }
}
