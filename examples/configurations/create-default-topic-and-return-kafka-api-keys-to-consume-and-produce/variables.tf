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
  description = "The ID of the Environment that the Kafka cluster belongs to of the form 'env-'"
  type        = string
}

variable "kafka_id" {
  description = "The ID of the Kafka cluster of the form 'lkc-'"
  type        = string
}

variable "topic_name" {
  description = "The name of the Kafka topic to create with default settings"
  type        = string
}
