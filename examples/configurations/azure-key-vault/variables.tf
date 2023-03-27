variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

# The Azure subscription ID to enable for the Private Link Access where your VNet exists.
# You can find your Azure subscription ID in the [Azure Portal on the Overview tab of your Azure Virtual Network](https://portal.azure.com/#blade/HubsExtension/BrowseResource/resourceType/Microsoft.Network%2FvirtualNetworks). Must be a valid 32 character UUID string.
variable "subscription_id" {
  description = "The Azure subscription ID to enable for the Private Link Access where your VNet exists"
  type        = string
  sensitive   = true
}

# The ID of the Client on Azure
# Follow Authenticating to Azure using a Service Principal and a Client Secret guide:
# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret#creating-a-service-principal
# to create Client ID and Client Secret
variable "client_id" {
  description = "The ID of the Client on Azure"
  type        = string
  sensitive   = true
}

variable "client_secret" {
  description = "The Secret of the Client on Azure"
  type        = string
  sensitive   = true
}

# The Azure tenant ID in which Subscription exists
# Represents an organization in Azure Active Directory. You can find your Azure Tenant ID in the [Azure Portal under Azure Active Directory](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/Overview). Must be a valid **32 character UUID string**.
variable "tenant_id" {
  description = "The Azure tenant ID in which Subscription exists"
  type        = string
  sensitive   = true
}
