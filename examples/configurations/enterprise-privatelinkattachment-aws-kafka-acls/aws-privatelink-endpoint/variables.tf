variable "vpc_id" {
  description = "The VPC ID to private link to Confluent Cloud"
  type = string
}

variable "privatelink_service_name" {
  description = "The Service Name from Confluent Cloud to Private Link with (provided by Confluent)"
  type = string
}

variable "bootstrap" {
  description = "The bootstrap server (ie: lkc-abcde-vwxyz.us-east-1.aws.glb.confluent.cloud:9092)"
  type = string
}

variable "subnets_to_privatelink" {
  description = "A map of Zone ID to Subnet ID (ie: {\"use1-az1\" = \"subnet-abcdef0123456789a\", ...})"
  type = map(string)
}

variable "dns_domain_name" {
  description = "The DNS domain name"
  type = string
}
