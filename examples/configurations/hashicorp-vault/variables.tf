variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "vault_address" {
  description = "Origin URL of the Vault server. This is a URL with a scheme, a hostname and a port but with no path."
  type        = string
}

variable "vault_token" {
  description = "Vault token that will be used by Terraform to authenticate."
  type        = string
  sensitive   = true
}