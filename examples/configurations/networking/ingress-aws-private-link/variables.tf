variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "The AWS region of the Ingress Private Link Gateway, for example, us-east-1"
  type        = string
}

variable "vpc_endpoint_id" {
  description = "ID of a VPC Endpoint that will be connected to the VPC Endpoint service, for example, vpce-00000000000000000"
  type        = string
}