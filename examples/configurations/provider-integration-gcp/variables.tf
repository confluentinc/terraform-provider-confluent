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

variable "environment_id" {
  description = "The ID of the Confluent Cloud environment"
  type        = string
}

variable "integration_display_name" {
  description = "Display name for the GCP provider integration"
  type        = string
  default     = "gcp-provider-integration"
}

variable "gcp_service_account" {
  description = "Your Google Service Account that Confluent Cloud will impersonate (e.g., my-sa@my-project.iam.gserviceaccount.com)"
  type        = string
}
