terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.13.0"
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

resource "confluent_network" "gcp-private-service-connect" {
  display_name     = "GCP Private Service Connect Network"
  cloud            = "GCP"
  region           = var.region
  connection_types = ["PRIVATELINK"]
  zones            = var.zones
  environment {
    id = confluent_environment.staging.id
  }

  dns_config {
    resolution = "PRIVATE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

data "confluent_gateway" "main" {
  id = confluent_network.gcp-private-service-connect.gateway[0].id
  environment {
    id = confluent_environment.staging.id
  }
  depends_on = [
    confluent_network.gcp-private-service-connect
  ]
}

resource "confluent_access_point" "private-service-connect" {
  display_name = "GCP Private Service Connect Access Point"
  environment {
    id = confluent_environment.staging.id
  }
  gateway {
    id = data.confluent_gateway.main.id
  }
  gcp_egress_private_service_connect_endpoint {
    private_service_connect_endpoint_target = "ALL_GOOGLE_APIS"
  }
  depends_on = [
    confluent_network.gcp-private-service-connect,
    data.confluent_gateway
  ]
}
