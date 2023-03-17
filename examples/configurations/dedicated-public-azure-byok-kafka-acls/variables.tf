variable "confluent_cloud_api_key" {
  type        = string
  description = "Cloud API key"
}

variable "confluent_cloud_api_secret" {
  type        = string
  description = "Cloud API secret"
  sensitive   = true
}

variable "subscription_id" {
  description = "The Azure subscription ID to enable for the Private Link Access where your VNet exists"
  type        = string
}

variable "client_id" {
  description = "The ID of the Client on Azure"
  type        = string
}

variable "client_secret" {
  description = "The Secret of the Client on Azure"
  type        = string
}

variable "tenant_id" {
  description = "The Azure tenant ID in which Subscription exists"
  type        = string
}
