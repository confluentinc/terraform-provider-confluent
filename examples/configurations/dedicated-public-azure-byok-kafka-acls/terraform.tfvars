# The Azure tenant ID in which Subscription exists
# Represents an organization in Azure Active Directory. You can find your Azure Tenant ID in the [Azure Portal under Azure Active Directory](https://portal.azure.com/#blade/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/Overview). Must be a valid **32 character UUID string**.
tenant_id = ""

# The Azure subscription ID to enable for the Private Link Access where your VNet exists
# You can find your Azure subscription ID in the [Azure Portal on the Overview tab of your Azure Virtual Network](https://portal.azure.com/#blade/HubsExtension/BrowseResource/resourceType/Microsoft.Network%2FvirtualNetworks). Must be a valid 32 character UUID string.
subscription_id = ""

# The ID of the Client on Azure
# Follow Authenticating to Azure using a Service Principal and a Client Secret guide:
# https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/guides/service_principal_client_secret#creating-a-service-principal
# to create Client ID and Client Secret
client_id = ""

# The Secret of the Client on Azure
client_secret = ""
