terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.4.0"
    }
    vault = {
      source  = "hashicorp/vault"
      version = "~> 3.14.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

provider "vault" {
  address          = var.vault_address
  token            = var.vault_token
  skip_child_token = true
}

resource "confluent_service_account" "hashicorp-vault" {
  display_name = "hashicorp-vault-service-account"
  description  = "Hashicorp Vault Example Service Account"
}

resource "confluent_environment" "staging" {
  display_name = "Staging"
}

resource "confluent_role_binding" "environment-admin" {
  principal   = "User:${confluent_service_account.hashicorp-vault.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = confluent_environment.staging.resource_name
}

resource "confluent_api_key" "main" {
  display_name = "hashicorp-vault-service-account-api-key"
  description  = "Cloud API Key for the Hashicorp Vault Example Service Account"
  owner {
    id          = confluent_service_account.hashicorp-vault.id
    api_version = confluent_service_account.hashicorp-vault.api_version
    kind        = confluent_service_account.hashicorp-vault.kind
  }

  depends_on = [confluent_role_binding.environment-admin]

  lifecycle {
    prevent_destroy = true
  }
}

resource "vault_kv_secret_v2" "main" {
  mount = "secret"
  name  = "sample-secret"

  data_json = jsonencode(
    {
      id     = confluent_api_key.main.id,
      secret = confluent_api_key.main.secret
    }
  )
}

# Uncomment for debugging
#data "vault_kv_secret_v2" "example" {
#  mount = vault_kv_secret_v2.main.mount
#  name  = vault_kv_secret_v2.main.name
#}
#
#output "kv_secret_cloud_api_key" {
#  value     = jsondecode(data.vault_kv_secret_v2.example.data_json)
#  sensitive = true
#}
#
#output "kv_secret_cloud_api_key_id" {
#  value     = jsondecode(data.vault_kv_secret_v2.example.data_json)["id"]
#  sensitive = true
#}
#
#output "kv_secret_cloud_api_key_secret" {
#  value     = jsondecode(data.vault_kv_secret_v2.example.data_json)["secret"]
#  sensitive = true
#}