output "service_principal_client_id" {
  description = "The Client ID (Application ID) of the Service Principal"
  value       = azuread_application.confluent_connector.client_id
}

output "service_principal_object_id" {
  description = "The Object ID of the Service Principal"
  value       = azuread_service_principal.confluent_connector.object_id
}

output "application_object_id" {
  description = "The Object ID of the Azure AD Application"
  value       = azuread_application.confluent_connector.object_id
}

output "tenant_id" {
  description = "Azure Tenant ID"
  value       = var.azure_tenant_id
}

