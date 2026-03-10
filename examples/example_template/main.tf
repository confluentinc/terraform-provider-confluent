terraform {
  required_providers {
    confluent = {
        source = "registry.terraform.io/confluentinc/confluent"
#       source  = "confluentinc/confluent"
#       version = "2.14.0"
    }
  }
}
provider "confluent" {
    cloud_api_key         = var.confluent_cloud_api_key
    cloud_api_secret      = var.confluent_cloud_api_secret
    endpoint = "https://api.stag.cpdev.cloud" # for staging environment
    # endpoint = "https://api.devel.cpdev.cloud" # for devel environment
}

# Add your resources/data sources here
# Eg.
# data "confluent_schemas" "main" {
# }