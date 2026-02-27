variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "aws_account_id" {
  description = "The AWS Account ID (12 digits)"
  type        = string
}

variable "region" {
  description = "The AWS Region"
  type        = string
}

variable "client_cidr_blocks" {
  description = "List of client CIDR blocks allowed to access EC2 via SSH"
  type        = list(string)
}
