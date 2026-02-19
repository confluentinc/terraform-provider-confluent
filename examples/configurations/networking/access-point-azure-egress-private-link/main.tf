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
  display_name = "azure-egress-private-link-gateway"
  environment {
    id = data.confluent_environment.main.id
  }
  azure_egress_private_link_gateway {
    region = var.region
  }
}

resource "confluent_access_point" "main" {
  display_name = "azure-egress-private-link-access-point"
  environment {
    id = data.confluent_environment.main.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  azure_egress_private_link_endpoint {
    private_link_service_resource_id = var.private_link_service_resource_id
    private_link_subresource_name    = var.private_link_subresource_name
  }
}
