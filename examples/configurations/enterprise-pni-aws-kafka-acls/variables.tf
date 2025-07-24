variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)."
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret."
  type        = string
  sensitive   = true
}

variable "aws_account_id" {
  description = "The AWS Account ID (12 digits)"
  type        = string
}

variable "region" {
  description = "The region of the AWS peer VPC."
  type        = string
}

variable "availability_zone_ids" {
  description = "List of 3 availability zone IDs"
  type        = list(string)
  validation {
    condition     = length(var.availability_zone_ids) == 3
    error_message = "Exactly 3 availability zone IDs must be provided."
  }
}

variable "num_eni_per_subnet" {
  description = "Number of ENIs to create per subnet"
  type        = number
  default     = 17
}