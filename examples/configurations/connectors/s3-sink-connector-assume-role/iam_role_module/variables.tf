variable "s3_bucket_name" {
  description = "The name of the S3 bucket"
  type        = string
  default     = "kostyatests4"
}

variable "provider_integration_external_id" {
  description = "The external ID of the Provider Integration. It's the unique external ID that Confluent Cloud uses when it assumes the IAM role in your Amazon Web Services (AWS) account."
  type        = string
}

variable "role_arn" {
  description = "The ARN of the IAM role that Confluent Cloud will assume in your AWS account. This role must have the necessary permissions to access the specified S3 bucket and must include the trust policy for the Confluent external ID."
  type        = string
}

variable "role_name" {
  description = "The name of the IAM role that Confluent Cloud will assume in your AWS account. This role must have the necessary permissions to access the specified S3 bucket and must include the trust policy for the Confluent external ID."
  type        = string
  default     = "ConfluentS3AccessRole"
}
