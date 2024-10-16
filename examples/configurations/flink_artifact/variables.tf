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
  description = "Path to .zip / .jar for Flink Artifact"
  type        = string
   # See "Create a User Defined Function for Flink SQL" here for more details
   # https://docs.confluent.io/cloud/current/flink/how-to-guides/create-udf.html#flink-sql-create-udf-upload-jar
  default     = "flink_artifact.jar"
}