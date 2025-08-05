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
  default     = ""
}

variable "oauth_identity_pool_id" {
  description = "The OAuth identity pool id from external identity provider, registered with Confluent Cloud"
  type        = string
  default     = "pool-W5Qe"
}

variable "west_kafka_cluster_id" {
  description = "ID of the 'west' Kafka Cluster (Public)"
  type        = string
  default     = "lkc-w359pg"
}

variable "west_kafka_cluster_environment_id" {
  description = "ID of the Environment that the 'west' Kafka Cluster (Public) belongs to"
  type        = string
  default     = "env-y2omyj"
}

variable "east_kafka_cluster_id" {
  description = "ID of the 'east' Kafka Cluster (Private)"
  type        = string
  default     = "lkc-rxd15k"
}

variable "east_kafka_cluster_environment_id" {
  description = "ID of the Environment that the 'east' Kafka Cluster (Private) belongs to"
  type        = string
  default     = "env-8y1wv7"
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
  default     = "advanced-bidirectional-link"
}
