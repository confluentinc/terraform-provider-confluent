variable "confluent_cloud_api_key" {
  type        = string
  description = "Cloud API key"
}

variable "confluent_cloud_api_secret" {
  type        = string
  description = "Cloud API secret"
  sensitive   = true
}
