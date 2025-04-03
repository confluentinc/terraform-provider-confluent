variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID) with EnvironmentAdmin and AccountAdmin roles provided by Kafka Ops team"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

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