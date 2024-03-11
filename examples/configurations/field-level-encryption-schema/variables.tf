variable "schema_registry_id" {
  description = "The ID of SR cluster (for example, `lsrc-abc123`)"
  type        = string
}

variable "schema_registry_rest_endpoint" {
  description = "The REST endpoint of SR cluster"
  type        = string
}

variable "schema_registry_api_key" {
  description = "Schema Registry API Key ID"
  type        = string
  sensitive   = true
}

variable "schema_registry_api_secret" {
  description = "Schema Registry API Key Secret"
  type        = string
  sensitive   = true
}

variable "aws_kms_key_arn" {
  description = "Key ID (ARN) of AWS KMS (for example, arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789abc)"
  type        = string
  sensitive   = true
}
