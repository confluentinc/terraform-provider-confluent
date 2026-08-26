variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_cluster_crn" {
  description = "The Confluent Resource Name (CRN) of the Kafka cluster the Client Request Policy is bound to."
  type        = string
}
