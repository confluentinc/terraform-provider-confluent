variable "oauth_external_token_url" {
  description = "The OAuth token URL from Microsoft Azure Entra ID as the identity provider"
  type        = string
}

variable "oauth_external_client_id" {
  description = "The OAuth token client id from Microsoft Azure Entra ID as the identity provider"
  type        = string
}

variable "oauth_external_client_secret" {
  description = "The OAuth token client secret from Microsoft Azure Entra ID as the identity provider"
  type        = string
  sensitive   = true
}

variable "oauth_external_token_scope" {
  description = "(Required field for Azure Entra ID) The OAuth client application scope from Microsoft Azure Entra ID as the identity provider"
  type        = string
}

variable "oauth_identity_pool_id" {
  description = "The OAuth identity pool id from Microsoft Azure Entra ID as the identity provider, registered with Confluent Cloud"
  type        = string
}