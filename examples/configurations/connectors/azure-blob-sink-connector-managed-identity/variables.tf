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

variable "azure_tenant_id" {
  description = "Azure Tenant ID"
  type        = string
}

variable "azure_subscription_id" {
  description = "Azure Subscription ID"
  type        = string
}

variable "azure_region" {
  description = "Azure region for Kafka cluster (e.g., eastus, westus2)"
  type        = string
  default     = "eastus"
}

variable "azure_storage_account_name" {
  description = "Azure Storage Account name"
  type        = string
}

variable "azure_resource_group_name" {
  description = "Azure Resource Group name where the storage account exists"
  type        = string
}

variable "azure_container_name" {
  description = "Azure Blob Storage container name"
  type        = string
}

