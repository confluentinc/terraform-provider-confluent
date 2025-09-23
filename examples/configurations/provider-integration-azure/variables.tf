variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
  sensitive   = true
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the Confluent Cloud environment"
  type        = string
}

variable "integration_display_name" {
  description = "Display name for the Azure provider integration"
  type        = string
  default     = "azure-provider-integration"
}

variable "azure_tenant_id" {
  description = "Your Azure Active Directory (Azure AD) tenant ID"
  type        = string
}
