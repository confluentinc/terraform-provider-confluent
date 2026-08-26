terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.83.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

resource "confluent_client_request_policy" "example" {
  name          = "restrict-client-versions"
  policy_type   = "VersionPolicy"
  resource_name = var.kafka_cluster_crn
  mode          = "AUDIT"
  action        = "DENY"

  rules {
    name  = "require-recent-java-client"
    match = "request.client.version >= '3.5.0'"
  }
}
