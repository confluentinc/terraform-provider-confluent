variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID) with CloudClusterAdmin permissions provided by Kafka Ops team"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_api_key" {
  description = "Kafka API Key with CloudClusterAdmin permissions provided by Kafka Ops team"
  type        = string
}

variable "kafka_api_secret" {
  description = "Kafka API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the managed environment"
  type        = string
}

variable "kafka_cluster_id" {
  description = "The ID of the managed Kafka cluster"
  type        = string
}

variable "app_manager_id" {
  description = "The ID of the app-manager service account"
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
