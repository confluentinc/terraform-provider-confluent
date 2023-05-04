variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "schema_namespace" {
  description = "The namespace of the schema"
  type        = string
}

variable "record_name" {
  description = "The name of the record in the schema"
  type        = string
}
