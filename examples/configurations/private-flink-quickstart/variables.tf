variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID) with EnvironmentAdmin and AccountAdmin roles provided by Kafka Ops team"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "environment_id" {
  description = "The ID of the managed environment"
  type        = string
}

variable "aws_account_id" {
  description = "The AWS Account ID (12 digits)"
  type        = string
}

variable "vpc_id" {
  description = "The ID of the VPC in which the endpoint will be used."
  type        = string
}

variable "region" {
  description = "The region of the AWS peer VPC."
  type        = string
}

variable "subnets_to_privatelink" {
  description = "A map of Zone ID to Subnet ID (i.e.: {\"use2-az2\" = \"subnet-000a2942a95976f38\", ...})"
  type        = map(string)
}

