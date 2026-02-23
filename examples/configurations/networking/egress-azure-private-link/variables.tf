variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "The Azure region of the Gateway, for example, eastus"
  type        = string
}

variable "private_link_service_resource_id" {
  description = "Resource ID of the Azure Private Link service"
  type        = string
}

variable "private_link_subresource_name" {
  description = "Name of the subresource for the Private Endpoint to connect to"
  type        = string
  default     = ""
}
