variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)."
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret."
  type        = string
}

variable "region" {
  description = "The region of Confluent Cloud Network."
  type        = string
}

variable "cidr" {
  description = "The CIDR of Confluent Cloud Network."
  type        = string
}

variable "aws_account_id" {
  description = "The AWS Account ID of the peer VPC owner (12 digits)."
  type        = string
}

variable "vpc_id" {
  description = "The AWS VPC ID of the peer VPC that you're peering with Confluent Cloud."
  type        = string
}

variable "routes" {
  description = "The AWS VPC CIDR blocks or subsets. This must be from the supported CIDR blocks and must not overlap with your Confluent Cloud CIDR block or any other network peering connection VPC CIDR."
  type        = list(string)
}

variable "customer_region" {
  description = "The region of the AWS peer VPC."
  type        = string
}
