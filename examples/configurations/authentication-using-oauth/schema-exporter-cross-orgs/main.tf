terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.65.0"
    }
  }
}

provider "confluent" {
  oauth {
    oauth_external_token_url     = var.oauth_external_token_url
    oauth_external_client_id     = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_external_token_scope   = var.oauth_external_token_scope
    oauth_identity_pool_id       = var.oauth_source_identity_pool_id
  }
}

resource "confluent_schema_exporter" "main" {
  schema_registry_cluster {
    id = var.source_schema_registry_cluster_id
  }

  name          = "my_exporter"
  rest_endpoint = var.source_schema_registry_rest_endpoint

  subjects = ["test-value"]

  destination_schema_registry_cluster {
    id            = var.destination_schema_registry_cluster_id
    rest_endpoint = var.destination_schema_registry_rest_endpoint
  }

  # TODO: consider adding sensitive_config {}
  config = {
    "bearer.auth.client.id" = "<client ID>"
    "bearer.auth.client.secret" = "<secret>"
    "bearer.auth.identity.pool.id" = var.oauth_destination_identity_pool_id
    "bearer.auth.scope" = "<client ID>/.default"
    "bearer.auth.credentials.source" = "OAUTHBEARER"
    "bearer.auth.issuer.endpoint.url" = "https://login.microsoftonline.com/<Azure Entra ID>/oauth2/v2.0/token"
  }

  reset_on_update = true
}
