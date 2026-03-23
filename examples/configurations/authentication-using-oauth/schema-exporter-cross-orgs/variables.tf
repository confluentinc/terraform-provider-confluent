variable "oauth_external_token_url" {
  description = "The OAuth token URL from the identity provider"
  type        = string
}

variable "oauth_external_client_id" {
  description = "The OAuth token client id from the identity provider"
  type        = string
}

variable "oauth_external_client_secret" {
  description = "The OAuth token client secret from the identity provider"
  type        = string
  sensitive   = true
}

variable "oauth_external_token_scope" {
  description = "(Required field for Azure Entra ID) The OAuth client application scope from the identity provider"
  type        = string
}

variable "oauth_source_identity_pool_id" {
  description = "The OAuth identity pool id from the identity provider, registered with Confluent Cloud (source org #1)"
  type        = string
}

variable "oauth_destination_identity_pool_id" {
  description = "The OAuth identity pool id from the identity provider, registered with Confluent Cloud (destination org #2)"
  type        = string
}
