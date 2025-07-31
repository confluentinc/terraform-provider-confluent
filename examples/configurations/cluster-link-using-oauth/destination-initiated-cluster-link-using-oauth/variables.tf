variable "oauth_external_token_url" {
  description = "The OAuth token URL from external identity provider"
  type        = string
}

variable "oauth_external_client_id" {
  description = "The OAuth token client id from external identity provider"
  type        = string
}

variable "oauth_external_client_secret" {
  description = "The OAuth token client secret from external identity provider"
  type        = string
  sensitive   = true
}

variable "oauth_identity_pool_id" {
  description = "The OAuth identity pool id from external identity provider, registered with Confluent Cloud"
  type        = string
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
}
