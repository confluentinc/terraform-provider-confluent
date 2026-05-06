variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)."
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret."
  type        = string
  sensitive   = true
}

variable "environment_name" {
  description = "The name of the Confluent Cloud environment to create."
  type        = string
}

variable "region" {
  description = "The Azure region for the gateway and cluster."
  type        = string
}

variable "resource_prefix" {
  description = "Prefix for resource display names to avoid collisions."
  type        = string
}

variable "private_endpoint_resource_id" {
  description = "Resource ID of the Azure Private Endpoint to connect to the Confluent ingress gateway. Example: /subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.Network/privateEndpoints/<pe-name>"
  type        = string
}
