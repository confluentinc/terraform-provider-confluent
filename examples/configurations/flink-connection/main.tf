terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.47.0"
    }
  }
}

provider "confluent" {
  organization_id       = var.organization_id            # optionally use CONFLUENT_ORGANIZATION_ID env var
  environment_id        = var.environment_id             # optionally use CONFLUENT_ENVIRONMENT_ID env var
  flink_compute_pool_id = var.flink_compute_pool_id      # optionally use FLINK_COMPUTE_POOL_ID env var
  flink_rest_endpoint   = var.flink_rest_endpoint        # optionally use FLINK_REST_ENDPOINT env var
  flink_api_key         = var.flink_api_key              # optionally use FLINK_API_KEY env var
  flink_api_secret      = var.flink_api_secret           # optionally use FLINK_API_SECRET env var
  flink_principal_id    = var.flink_principal_id         # optionally use FLINK_PRINCIPAL_ID env var
}

resource "confluent_flink_connection" "connection" {
      display_name = "connection"
      type = "OPENAI"
      endpoint = "https://api.openai.com/v1/chat/completions"
      api_key ="0000000000000001"
}
