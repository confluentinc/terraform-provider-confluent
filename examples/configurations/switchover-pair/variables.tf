variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also configurable via the CONFLUENT_CLOUD_API_KEY environment variable)."
  type        = string
  sensitive   = true
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret (also configurable via the CONFLUENT_CLOUD_API_SECRET environment variable)."
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The environment ID the switchover pair belongs to (e.g. env-abc123)."
  type        = string
}

variable "west_cluster_id" {
  description = "The Kafka cluster ID for the 'west' member (e.g. lkc-111111)."
  type        = string
}

variable "east_cluster_id" {
  description = "The Kafka cluster ID for the 'east' member (e.g. lkc-222222)."
  type        = string
}
