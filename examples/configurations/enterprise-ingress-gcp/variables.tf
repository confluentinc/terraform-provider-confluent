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
  description = "The GCP region for the gateway and cluster."
  type        = string
}

variable "resource_prefix" {
  description = "Prefix for resource display names to avoid collisions."
  type        = string
}

variable "private_service_connect_connection_id" {
  description = "The ID of the GCP Private Service Connect connection. Create this in GCP first, connecting to the service_attachment from the gateway output."
  type        = string
}
