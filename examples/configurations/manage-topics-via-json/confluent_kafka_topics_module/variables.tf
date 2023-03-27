variable "cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_api_key" {
  description = "Kafka API Key that is owned by service account that has permissions to create topics (e.g., has at least CloudClusterAdmin role)"
  type        = string
  sensitive   = true
}

variable "kafka_api_secret" {
  description = "Kafka API Secret"
  type        = string
  sensitive   = true
}

variable "kafka_rest_endpoint" {
  description = "The REST Endpoint of the Kafka cluster"
  type        = string
}

variable "kafka_id" {
  description = "The ID the the Kafka cluster of the form 'lkc-'"
  type        = string
}

variable "topics" {
  description = "A map of Kafka topic configurations"
  type        = map(object({ config : map(string), partitions_count : number }))
  default     = {}
}
