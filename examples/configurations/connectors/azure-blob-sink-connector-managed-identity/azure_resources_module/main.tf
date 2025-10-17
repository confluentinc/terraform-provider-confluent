# https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html
# This module creates the necessary Azure resources for Confluent Cloud to access your Azure Blob Storage
# using Managed Identity authentication with Federated Identity Credentials (Workload Identity Federation).

# Create Azure AD Application
resource "azuread_application" "confluent_connector" {
  display_name = var.application_name
}

# Create a Service Principal (the identity that will be granted access)
resource "azuread_service_principal" "confluent_connector" {
  client_id = azuread_application.confluent_connector.client_id
}

# Allow Confluent identity to impersonate the service principal via Federated Identity Credential
# This establishes OIDC trust between Confluent Cloud and Azure AD
resource "azuread_application_federated_identity_credential" "confluent_trust" {
  application_id = azuread_application.confluent_connector.id
  display_name   = "confluent-oidc-trust"
  description    = "OIDC trust for Confluent to impersonate this Service Principal"
  issuer         = "https://token.confluent.cloud"
  
  # Subject format: system:serviceaccount:confluent:<connector-name>
  # This should match the connector's identity in Confluent Cloud
  subject = "system:serviceaccount:confluent:azure-blob-sink-connector"
  
  # Standard audience for Azure Workload Identity Federation
  audiences = ["api://AzureADTokenExchange"]
}

# Lookup the storage account
data "azurerm_storage_account" "target" {
  name                = var.storage_account_name
  resource_group_name = var.resource_group_name
}

# Lookup the blob container
data "azurerm_storage_container" "target" {
  name                 = var.container_name
  storage_account_name = data.azurerm_storage_account.target.name
}

# Assign Storage Blob Data Contributor role to the Service Principal on the container
# This allows the connector to write blobs to the container
resource "azurerm_role_assignment" "blob_contributor" {
  scope                = data.azurerm_storage_container.target.resource_manager_id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azuread_service_principal.confluent_connector.object_id
}

