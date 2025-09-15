output "provider_integration_id" {
  description = "The ID of the created provider integration"
  value       = confluent_provider_integration_v2.azure.id
}

output "provider_integration_status" {
  description = "The status of the provider integration"
  value       = confluent_provider_integration_v2.azure.status
}

output "confluent_multi_tenant_app_id" {
  description = "The Confluent Multi-Tenant App ID for Azure access"
  value       = confluent_provider_integration_v2_authorization.azure.azure[0].confluent_multi_tenant_app_id
}

output "azure_setup_command" {
  description = "Azure CLI command to create the service principal"
  value       = "az ad sp create --id ${confluent_provider_integration_v2_authorization.azure.azure[0].confluent_multi_tenant_app_id}"
}

output "azure_admin_consent_url" {
  description = "URL to grant admin consent for the multi-tenant app"
  value       = "https://login.microsoftonline.com/${var.azure_tenant_id}/adminconsent?client_id=${confluent_provider_integration_v2_authorization.azure.azure[0].confluent_multi_tenant_app_id}"
}