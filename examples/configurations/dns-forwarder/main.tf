terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.9.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  endpoint         = "https://api.stag.cpdev.cloud"
}

resource "confluent_dns_forwarder" "main" {
  display_name = "dns_forwarder"
  environment {
    id = var.confluent_environment_id
  }
  domains = ["example.com"]
  gateway {
    id = var.confluent_gateway_id
  }

  forward_via_gcp_dns_zones {
    domain_mappings = {
      "example.com" = "us-central1-a,cc-stag"
    }
  }
}