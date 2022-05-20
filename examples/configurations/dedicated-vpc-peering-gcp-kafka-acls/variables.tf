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
  description = "The region of Confluent Cloud Network"
  type        = string
}

variable "cidr" {
  description = "The CIDR of Confluent Cloud Network"
  type        = string
}

variable "customer_region" {
  description = "The region of the GCP peer VPC network"
  type        = string
}

variable "customer_project_id" {
  description = "The GCP Project ID"
  type        = string
}

variable "customer_vpc_network" {
  description = "The VPC network name that you're peering to Confluent Cloud"
  type        = string
}

variable "customer_peering_name" {
  description = "The name of the peering on GCP that will be created via TF"
  type        = string
}

variable "import_custom_routes" {
  description = "The Import Custom Routes option enables connectivity to a Confluent Cloud cluster in Google Cloud from customer premise or other clouds, such as AWS and Azure, through a customer VPC that is peered with Confluent Cloud in the same region."
  type        = bool
  default     = false
}
