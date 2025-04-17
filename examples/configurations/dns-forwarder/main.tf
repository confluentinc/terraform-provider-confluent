terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.25.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "staging" {
  display_name = "Staging10"

  stream_governance {
    package = "ESSENTIALS"
  }
}

resource "confluent_network" "gcp-peering" {
  display_name     = "GCP Peering Network"
  cloud            = "GCP"
  region           = var.region
  connection_types = ["PEERING"]
  zones            = var.zones
  environment {
    id = confluent_environment.staging.id
  }
  cidr             = "192.168.0.0/16"

  lifecycle {
    prevent_destroy = true
  }
}

data "confluent_gateway" "main" {
  id = confluent_network.gcp-peering.gateway[0].id
  environment {
    id = confluent_environment.staging.id
  }
  depends_on = [
    confluent_network.gcp-peering
  ]
}

resource "confluent_dns_forwarder" "main" {
  display_name = "dns_forwarder"
  environment {
    id = confluent_environment.staging.id
  }
  domains = ["example.com"]
  gateway {
    id = data.confluent_gateway.main.id
  }
  forward_via_gcp_dns_zones {
    domain_mappings = {
      "example.com" = "us-central1-a,cc-stag"
    }
  }
}
