terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.12.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  endpoint = "stag.cpdev.cloud"
}

resource "confluent_environment" "development" {
  display_name = "Stag_GCP_OPT_Test"

  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_network" "gcp-private-service-connect" {
  display_name     = "GCP Private Service Connect Network"
  cloud            = "GCP"
  region           = "us-central1"
  connection_types = ["PRIVATELINK"]
  zones            = ["us-central1-a", "us-central1-b", "us-central1-c"]
  environment {
    id = confluent_environment.development.id
  }

  dns_config {
    resolution = "PRIVATE"
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_access_point" "gcp-private-access-point" {
  display_name = "gcp_access_point"
  environment {
    id = confluent_environment.development.id
  }
  gateway {
    id = confluent_network.gcp-private-service-connect.gateway[0].id
  }
  gcp_egress_private_link_endpoint {
    private_service_connect_endpoint_target = "all-google-apis"
  }
  depends_on = [
    confluent_network.gcp-private-service-connect
  ]
  lifecycle {
    prevent_destroy = true
  }
}