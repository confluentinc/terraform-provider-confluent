variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "artifact_file" {
  description = "Path to .zip / .jar for Connect Artifact"
  type        = string
}

variable "organization_id" {
  description = "The ID of Confluent Cloud organization (for example, foobar). You could find it on XYZ page."
  type        = string
}

variable "environment_id" {
  description = "The ID of the managed environment on Confluent Cloud."
  type        = string
}

variable "kafka_cluster_id" {
  description = "The ID of the managed Kafka Cluster on Confluent Cloud."
  type        = string
}

variable "kafka_api_key" {
  description = "Kafka API Key for the connector to authenticate with the Kafka cluster."
  type        = string
}

variable "kafka_api_secret" {
  description = "Kafka API Secret for the connector to authenticate with the Kafka cluster."
  type        = string
  sensitive   = true
}

variable "artifact_display_name" {
  description = "Display name for the Connect Artifact"
  type        = string
  default     = "example-connect-artifact"
}

variable "artifact_description" {
  description = "Description for the Connect Artifact"
  type        = string
  default     = "The description"
}

variable "artifact_content_format" {
  description = "Content format of the artifact (JAR or ZIP)"
  type        = string
  default     = "JAR"
}

variable "artifact_cloud" {
  description = "Cloud provider for the artifact"
  type        = string
  default     = "AWS"
} 