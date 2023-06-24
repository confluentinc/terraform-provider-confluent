variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
  default = "YW4YATHNHKWLFAV4"
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
  default = "No0cNdfhjDyX2lBE2apmyuR04NGHqmCcW2C5HemlGgu2/Ls+VpqBACmX5n+ai5W2"
}


variable "region" {
  description = "The region of Confluent Cloud Network."
  type        = string
  default = "us-west-2"
}

variable "cidr" {
  description = "The CIDR of Confluent Cloud Network."
  type        = string
  default = "192.168.0.0/16"
}

variable "aws_account_id" {
  description = "The AWS Account ID of the peer VPC owner (12 digits)."
  type        = string
  default = "098044532734"
}

variable "vpc_id" {
  description = "The AWS VPC ID of the peer VPC that you're peering with Confluent Cloud."
  type        = string
  default = "vpc-09b13241533f988f9"
}

variable "routes" {
  description = "The AWS VPC CIDR blocks or subsets. This must be from the supported CIDR blocks and must not overlap with your Confluent Cloud CIDR block or any other network peering connection VPC CIDR."
  type        = list(string)
  default = [ "10.0.0.0/16" ]
}

variable "customer_region" {
  description = "The region of the AWS peer VPC."
  type        = string
  default = "us-west-2"
}
