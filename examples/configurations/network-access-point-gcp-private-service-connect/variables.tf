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
  description = "The region of Confluent Cloud Network"
  type        = string
}

variable "zones" {
  description = "The 3 availability GCP zones for this Confluent Cloud Network"
  type = list(string)
}
