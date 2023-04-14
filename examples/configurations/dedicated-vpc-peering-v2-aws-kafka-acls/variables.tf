variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)."
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret."
  type        = string
  sensitive   = true
}

variable "region" {
  description = "The region of Confluent Cloud Network."
  type        = string
}

variable "zones_info" {
  description = "The Confluent Cloud Network Availability Zones Metadata. Each item represents information related to a single zone."

  type = list(object({
    // Cloud provider availability zone id (e.g., use1-az1)
    zone_id = string,
    // The IPv4 CIDR block to used for this network. Must be a /27. Required for VPC Peering and AWS Transit Gateway.
    cidr = string
  }))
}

variable "aws_account_id" {
  description = "The AWS Account ID of the peer VPC owner (12 digits)."
  type        = string
}

variable "vpc_id" {
  description = "The AWS VPC ID of the peer VPC that you're peering with Confluent Cloud."
  type        = string
}

variable "customer_region" {
  description = "The region of the AWS peer VPC."
  type        = string
}
