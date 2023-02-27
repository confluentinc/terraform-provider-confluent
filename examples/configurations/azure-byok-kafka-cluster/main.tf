terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.32.0"
    }

    azurerm = {
      source  = "hashicorp/azurerm"
      version = "=3.44.1"
    }

    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.15.0"
    }
  }
}

provider "azurerm" {
  features {}
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

data "azuread_client_config" "current" {}
data "azurerm_client_config" "current" {}

# Create a key vault
resource "azurerm_resource_group" "main" {
  name     = "keyvault-rg"
  location = "Central US"
}

resource "azurerm_key_vault" "main" {
  name                        = "byok-keyvault-terraform"
  location                    = azurerm_resource_group.main.location
  resource_group_name         = azurerm_resource_group.main.name
  enabled_for_disk_encryption = true
  tenant_id                   = data.azurerm_client_config.current.tenant_id
  soft_delete_retention_days  = 7
  purge_protection_enabled    = true
  enable_rbac_authorization   = true

  sku_name = "standard"

}

resource "azurerm_role_assignment" "administrator_assignment" {
  scope                = azurerm_key_vault.main.id
  role_definition_name = "Key Vault Administrator"
  principal_id         = data.azuread_client_config.current.object_id
}

# Create an Azure Key
resource "azurerm_key_vault_key" "main" {
  name         = "byok-key"
  key_vault_id = azurerm_key_vault.main.id
  key_type     = "RSA"
  key_size     = 2048

  key_opts = [
    "decrypt",
    "encrypt",
    "sign",
    "unwrapKey",
    "verify",
    "wrapKey",
  ]
}

# Create Confluent Cloud BYOK key
resource "confluent_byok_key" "main" {
  azure {
    tenant_id      = data.azurerm_client_config.current.tenant_id
    key_vault_id   = azurerm_key_vault.main.id
    key_identifier = azurerm_key_vault_key.main.versionless_id
  }
}

# Create service principal referencing the application ID returned by the confluent cloud key
resource "azuread_service_principal" "main" {
  application_id               = confluent_byok_key.main.azure[0].application_id
  app_role_assignment_required = false
  owners                       = [data.azuread_client_config.current.object_id]
}

# Create role assignments to the service principal to allow Confluent access to the keyvault
resource "azurerm_role_assignment" "reader_role_assignment" {
  scope                = confluent_byok_key.main.azure[0].key_vault_id
  role_definition_name = "Key Vault Reader"
  principal_id         = azuread_service_principal.main.object_id
}

resource "azurerm_role_assignment" "encryption_user_role_assignment" {
  scope                = confluent_byok_key.main.azure[0].key_vault_id
  role_definition_name = "Key Vault Crypto Service Encryption User"
  principal_id         = azuread_service_principal.main.object_id
}

# Create Kafka Cluster
resource "confluent_environment" "development" {
  display_name = "Development"

}

resource "confluent_kafka_cluster" "dedicated_byok" {
  display_name = "byok_kafka_cluster"
  availability = "SINGLE_ZONE"
  cloud        = "AZURE"
  region       = "centralus"
  dedicated {
    cku = 1
  }

  environment {
    id = confluent_environment.development.id
  }

  byok_key {
    id = confluent_byok_key.main.id
  }
}
