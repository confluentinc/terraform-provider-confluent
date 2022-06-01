variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID) with EnvironmentAdmin permissions provided by Kafka Ops team"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the managed environment"
  type        = string
}

variable "env_manager_id" {
  description = "The ID of the env-manager service account"
  type        = string
}

variable "app_producer_id" {
  description = "The ID of the app-producer service account"
  type        = string
}

variable "app_consumer_id" {
  description = "The ID of the app-consumer service account"
  type        = string
}
