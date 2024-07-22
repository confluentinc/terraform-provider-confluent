variable "resource_group" {
  description = "Resource group of the VNET"
  type        = string
}

variable "vnet_region" {
  description = "The Azure Region of the existing VNET"
  type        = string
}

variable "vnet_name" {
  description = "The VNET Name to private link to Confluent Cloud"
  type        = string
}

variable "dns_domain" {
  description = "The root DNS domain for the Private Link Attachment, for example, `pr123a.us-east-2.aws.confluent.cloud`"
  type        = string
}

variable "private_link_service_alias" {
  description = "Private link_service_alias"
  type        = string
}

variable "subnet_name_by_zone" {
  description = "A map of Zone to Subnet Name"
  type        = map(string)
}
