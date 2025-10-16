variable "customer_vpc_network" {
  description = "The VPC network name that you're peering to Confluent Cloud"
  type        = string
}

variable "dns_domain" {
  description = "The root DNS domain for the Private Link Attachment, for example, `pr123a.us-east-2.aws.confluent.cloud`"
  type        = string
}

variable "customer_subnetwork_name" {
  description = "The subnetwork name to provision Private Service Connect endpoint to Confluent Cloud"
  type        = string
}

variable "platt_service_attachment_uri" {
  description = "The Service attachment URI for the PrivateLink Attachment gateway in the Confluent Cloud"
  type        = string
}
