variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "confluent_gateway_id" {
  description = "Network Gateway ID"
  type        = string
}

variable "confluent_environment_id" {
  description = "Environment ID"
  type        = string
}

