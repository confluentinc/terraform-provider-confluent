variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "destination_kafka_cluster_id" {
  description = "ID of the Destination Kafka Cluster"
  type        = string
}

variable "destination_kafka_cluster_environment_id" {
  description = "ID of the Environment that the Destination Kafka Cluster belongs to"
  type        = string
}

variable "source_kafka_cluster_id" {
  description = "ID of the Source Kafka Cluster"
  type        = string
}

variable "source_kafka_cluster_environment_id" {
  description = "ID of the Environment that the Source Kafka Cluster belongs to"
  type        = string
}

variable "source_topic_name" {
  description = "Name of the Topic on the Source Kafka Cluster to create a Mirror Topic for"
  type        = string
}

variable "cluster_link_name" {
  description = "Name of the Cluster Link to create"
  type        = string
  default     = "destination-initiated-cluster-link-terraform"
}
