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
  description = "The AWS region of the Gateway, for example, us-east-1"
  type        = string
}

variable "vpc_endpoint_service_name" {
  description = "AWS VPC Endpoint Service Name, for example, com.amazonaws.vpce.us-west-2.vpce-svc-0d3be37e21708ecd3"
  type        = string
}

variable "enable_high_availability" {
  description = "Whether the Access Point should be provisioned with high availability"
  type        = bool
  default     = false
}
