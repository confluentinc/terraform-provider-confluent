variable "s3_bucket_name" {
  description = "The name of the S3 bucket"
  type        = string
}

variable "provider_integration_external_id" {
  description = "The external ID of the Provider Integration. It's the unique external ID that Confluent Cloud uses when it assumes the IAM role in your Amazon Web Services (AWS) account."
  type        = string
}

variable "provider_integration_role_arn" {
  description = "The IAM role ARN used in Confluent Cloud internally, bundled with customer_role_arn."
  type        = string
}

variable "customer_role_name" {
  description = "The name of the IAM role for accessing S3 with a trust policy for Confluent"
  type        = string
}
