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
  display_name = "aws-private-network-interface-gateway"
  environment {
    id = confluent_environment.staging.id
  }
  aws_private_network_interface_gateway {
    region = var.region
    zones  = var.availability_zone_ids
  }
}

resource "confluent_access_point" "main" {
  display_name = "aws-private-network-interface-access-point"
  environment {
    id = confluent_environment.staging.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_private_network_interface {
    network_interfaces = var.network_interface_ids
    account            = var.aws_account_id
  }
  depends_on = [
    confluent_gateway.main
  ]
}
