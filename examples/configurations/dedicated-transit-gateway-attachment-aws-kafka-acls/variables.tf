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

variable "cidr" {
  description = "The CIDR of Confluent Cloud Network."
  type        = string
}

variable "aws_account_id" {
  description = "The AWS Account ID of the VPC owner (12 digits)."
  type        = string
}

variable "vpc_id" {
  description = "The AWS VPC ID of the VPC that you're connecting with Confluent Cloud."
  type        = string
}

variable "transit_gateway_id" {
  description = "The AWS Transit Gateway ID of the VPC that you're connecting with Confluent Cloud."
  type        = string
}

variable "customer_region" {
  description = "The region of the AWS VPC."
  type        = string
}

variable "routes" {
  description = "The AWS VPC CIDR blocks or subsets. List of destination routes for traffic from Confluent VPC to your VPC via Transit Gateway."
  type        = list(string)
  default     = ["100.64.0.0/10", "10.0.0.0/8", "192.168.0.0/16", "172.16.0.0/12"]
}
