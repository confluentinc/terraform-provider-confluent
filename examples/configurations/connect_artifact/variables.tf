variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the managed environment on Confluent Cloud."
  type        = string
}

variable "artifact_file" {
  description = "Path to the connector artifact file (JAR or ZIP)"
  type        = string
  default     = "connect-artifact.jar"
} 