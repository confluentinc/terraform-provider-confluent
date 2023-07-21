variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "west_kafka_cluster_id" {
  description = "ID of the 'west' Kafka Cluster"
  type        = string
}

variable "west_kafka_cluster_environment_id" {
  description = "ID of the Environment that the 'west' Kafka Cluster belongs to"
  type        = string
}

variable "east_kafka_cluster_id" {
  description = "ID of the 'east' Kafka Cluster"
  type        = string
}

variable "east_kafka_cluster_environment_id" {
  description = "ID of the Environment that the 'east' Kafka Cluster belongs to"
  type        = string
}

variable "east_topic_name" {
  description = "Name of the Topic on the 'east' Kafka Cluster to create a Mirror Topic for"
  type        = string
  default     = "public.topic-on-east"
}

variable "west_topic_name" {
  description = "Name of the Topic on the 'west' Kafka Cluster to create a Mirror Topic for"
  type        = string
  default     = "public.topic-on-west"
}

variable "cluster_link_name" {
  description = "Name of the Cluster Link to create"
  type        = string
  default     = "bidirectional-link"
}
