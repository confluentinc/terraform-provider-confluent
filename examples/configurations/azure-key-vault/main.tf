terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.36.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0.2"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

provider "azurerm" {
  features {}
  subscription_id = var.subscription_id
  client_id       = var.client_id
  client_secret   = var.client_secret
  tenant_id       = var.tenant_id
}

resource "confluent_service_account" "azure-keyvault" {
  display_name = "azure-keyvault-service-account"
  description  = "Azure Key Vault Example Service Account"
}

resource "confluent_environment" "staging" {
  display_name = "Staging"
}

resource "confluent_role_binding" "environment-admin" {
  principal   = "User:${confluent_service_account.azure-keyvault.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = confluent_environment.staging.resource_name
}

resource "confluent_api_key" "main" {
  display_name = "azure-keyvault-service-account-api-key"
  description  = "Cloud API Key for the Azure Key Vault Example Service Account"
  owner {
    id          = confluent_service_account.azure-keyvault.id
    api_version = confluent_service_account.azure-keyvault.api_version
    kind        = confluent_service_account.azure-keyvault.kind
  }

  depends_on = [confluent_role_binding.environment-admin]

  lifecycle {
    prevent_destroy = true
  }
}

resource "azurerm_resource_group" "rg" {
  name     = "confluent-example-rg"
  location = "centralus"
}

data "azurerm_client_config" "current" {}
resource "azurerm_key_vault" "keyvault" {
  depends_on                  = [azurerm_resource_group.rg]
  name                        = "confluent-example-kv"
  location                    = azurerm_resource_group.rg.location
  resource_group_name         = azurerm_resource_group.rg.name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = false

  sku_name = "standard"

  access_policy {
    tenant_id = data.azurerm_client_config.current.tenant_id
    object_id = data.azurerm_client_config.current.object_id

    key_permissions = [
      "Get",
    ]

    secret_permissions = [
      "Get", "Backup", "Delete", "List", "Purge", "Recover", "Restore", "Set",
    ]

    storage_permissions = [
      "Get",
    ]
  }
}

resource "azurerm_key_vault_secret" "main" {
  name = confluent_service_account.azure-keyvault.display_name
  value = jsonencode(
    {
      id     = "${confluent_api_key.main.id}",
      secret = "${confluent_api_key.main.secret}"
    }
  )
  key_vault_id = azurerm_key_vault.keyvault.id
  depends_on   = [azurerm_key_vault.keyvault]
}

# Uncomment for debugging
#data "azurerm_key_vault_secret" "example" {
#  name         = confluent_service_account.azure-keyvault.display_name
#  key_vault_id = azurerm_key_vault.keyvault.id
#}
#
#output "key_vault_secret_cloud_api_key" {
#  value     = jsondecode(data.azurerm_key_vault_secret.example.value)
#  sensitive = true
#}
#
#output "key_vault_secret_cloud_api_key_id" {
#  value     = jsondecode(data.azurerm_key_vault_secret.example.value)["id"]
#  sensitive = true
#}
#
#output "key_vault_secret_cloud_api_key_secret" {
#  value     = jsondecode(data.azurerm_key_vault_secret.example.value)["secret"]
#  sensitive = true
#}
