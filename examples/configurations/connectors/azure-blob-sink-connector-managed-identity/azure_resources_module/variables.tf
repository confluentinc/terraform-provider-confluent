variable "application_name" {
  description = "Display name for the Azure AD Application"
  type        = string
}

variable "confluent_multi_tenant_app_id" {
  description = "Confluent Multi-Tenant App ID returned from the provider integration authorization"
  type        = string
}

variable "azure_tenant_id" {
  description = "Azure Tenant ID"
  type        = string
}

variable "storage_account_name" {
  description = "Azure Storage Account name"
  type        = string
}

variable "resource_group_name" {
  description = "Azure Resource Group name where the storage account exists"
  type        = string
}

variable "container_name" {
  description = "Azure Blob Storage container name"
  type        = string
}

