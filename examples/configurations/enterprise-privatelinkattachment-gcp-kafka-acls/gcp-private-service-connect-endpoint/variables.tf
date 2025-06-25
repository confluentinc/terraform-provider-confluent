variable "customer_vpc_network" {
  description = "The VPC network name to provision Private Service Connect endpoint to Confluent Cloud"
  type        = string
}

variable "customer_subnetwork_name" {
  description = "The subnetwork name to provision Private Service Connect endpoint to Confluent Cloud"
  type        = string
}

variable "subnet_name_by_zone" {
  description = "A map of Zone to Subnet Name"
  type        = map(string)
}

variable "dns_domain" {
  description = "The root DNS domain for the Private Link Attachment, for example, `pr123a.us-east-2.aws.confluent.cloud`"
  type        = string
}

variable "privatelink_service_name" {
  description = "The Service Name from Confluent Cloud to Private Link with (provided by Confluent)"
  type        = string
}