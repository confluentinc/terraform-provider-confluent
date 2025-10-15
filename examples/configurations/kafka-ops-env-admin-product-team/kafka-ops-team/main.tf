terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.49.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

data "confluent_organization" "this" {}

resource "confluent_environment" "staging" {
  display_name = "Staging"
}

resource "confluent_service_account" "env-manager" {
  display_name = "env-manager"
  description  = "Service account to manage resources under 'Staging' environment"
}

resource "confluent_role_binding" "env-manager-env-admin" {
  principal   = "User:${confluent_service_account.env-manager.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = confluent_environment.staging.resource_name
}

resource "confluent_role_binding" "env-manager-account-admin" {
  principal   = "User:${confluent_service_account.env-manager.id}"
  role_name   = "AccountAdmin"
  crn_pattern = data.confluent_organization.this.resource_name
}

resource "confluent_api_key" "env-manager-cloud-api-key" {
  display_name = "env-manager-cloud-api-key"
  description  = "Cloud API Key to be shared with Product team to manage resources under 'Staging' environment"
  owner {
    id          = confluent_service_account.env-manager.id
    api_version = confluent_service_account.env-manager.api_version
    kind        = confluent_service_account.env-manager.kind
  }

  depends_on = [
    confluent_role_binding.env-manager-env-admin,
    confluent_role_binding.env-manager-account-admin
  ]
}
