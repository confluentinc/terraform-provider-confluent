variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "s3_bucket_name" {
  description = "The name of the S3 bucket. S3 buckets must be in the same region as the cluster"
  type        = string
  default     = "myuswest2bucket2"
}

variable "aws_region" {
  description = "The AWS region where the S3 bucket is located."
  type        = string
  default     = "us-west-2"
}