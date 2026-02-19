variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the Confluent Cloud Environment"
  type        = string
}

variable "region" {
  description = "The AWS region of the Private Network Interface Gateway, for example, us-east-1"
  type        = string
}

variable "availability_zone_ids" {
  description = "The AWS availability zone IDs for the Private Network Interface Gateway, for example, [\"use1-az1\", \"use1-az2\", \"use1-az4\"]"
  type        = list(string)
}

variable "network_interface_ids" {
  description = "List of the IDs of the Elastic Network Interfaces"
  type        = list(string)
}

variable "aws_account_id" {
  description = "The AWS account ID associated with the ENIs, for example, 000000000000"
  type        = string
}
