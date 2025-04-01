variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID) with EnvironmentAdmin and AccountAdmin roles provided by Kafka Ops team"
  type        = string
  default     = "VBMLJNNQSG7LHEVN"
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
  default     = "bnVeT4W1wBtxgaNIZ6WjdtdT1yoRS08NjBXv+LIc5yH7aS6qM+jZetBXQMJTjBJ+"
}

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