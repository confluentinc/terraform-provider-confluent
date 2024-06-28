variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "custom_connector_plugin_filename" {
  description = "Path to .zip / .jar for Datagen Source Connector"
  type        = string
  # See "Getting a connector" section at
  # https://docs.confluent.io/cloud/current/connectors/bring-your-connector/custom-connector-qs.html#getting-a-connector
  default = "confluentinc-kafka-connect-datagen-0.6.2.zip"
}
