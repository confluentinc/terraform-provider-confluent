variable "oauth_external_token_url" {
  description = "The OAuth token URL from external identity provider"
  type        = string
  default     = "https://ccloud-sso-sandbox.okta.com/oauth2/ausod37qoaxy2xfjI697/v1/token"
}

variable "oauth_external_client_id" {
  description = "The OAuth token client id from external identity provider"
  type        = string
  default     = "0oaod69tu7yYnbrMn697"
}

variable "oauth_external_client_secret" {
  description = "The OAuth token client secret from external identity provider"
  type        = string
  sensitive   = true
  default     = "QN83b_1JscAAep7JTEdXvdloEUqKBXwk4_K00VyzXnYYBVFbCxdZN4Vy6NUDdo04"
}

variable "oauth_identity_pool_id" {
  description = "The OAuth identity pool id from external identity provider, registered with Confluent Cloud"
  type        = string
  default     = "pool-W5Qe"
}

variable "destination_kafka_cluster_id" {
  description = "ID of the Destination Kafka Cluster"
  type        = string
  default     = "lkc-x81kwg"
}

variable "destination_kafka_cluster_environment_id" {
  description = "ID of the Environment that the Destination Kafka Cluster belongs to"
  type        = string
  default     = "env-y2omyj"
}

variable "source_kafka_cluster_id" {
  description = "ID of the Source Kafka Cluster"
  type        = string
  default     = "lkc-og066y"
}

variable "source_kafka_cluster_environment_id" {
  description = "ID of the Environment that the Source Kafka Cluster belongs to"
  type        = string
  default     = "env-8y1wv7"
}

variable "source_topic_name" {
  description = "Name of the Topic on the Source Kafka Cluster to create a Mirror Topic for"
  type        = string
  default     = "cluster_link_topic_test1"
}
