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
  display_name = "aws-private-network-interface-gateway"
  environment {
    id = data.confluent_environment.main.id
  }
  aws_private_network_interface_gateway {
    region = var.region
    zones  = var.availability_zone_ids
  }
}

resource "confluent_access_point" "main" {
  display_name = "aws-private-network-interface-access-point"
  environment {
    id = data.confluent_environment.main.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_private_network_interface {
    network_interfaces = var.network_interface_ids
    account            = var.aws_account_id
  }
}
