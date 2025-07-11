terraform {
  required_providers {
    confluent = {
      # source  = "confluentinc/confluent"
      # version = "2.29.0"
      source = "terraform.confluent.io/confluentinc/confluent"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  endpoint         = "https://api.stag.cpdev.cloud"
}

resource "confluent_environment" "development" {
  display_name = "pni-tf-testing"
}

resource "confluent_gateway" "main" {
  display_name = "my_gateway"
  environment {
    id = confluent_environment.development.id
  }
  aws_private_network_interface_gateway {
    region = "us-west-2"
    zones = ["usw2-az1", "usw2-az2", "usw2-az3"]
  }
}


resource "confluent_access_point" "aws" {
  display_name = "access_point_2"
  environment {
    id = confluent_environment.development.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_private_network_interface {
    network_interfaces = [
      "eni-0cf3e827666b8b0a7","eni-0a365dc7df29e9225","eni-03a38e3b8ef37c0f9",
      "eni-0a291a4d82b5433f9","eni-0b4358b13af86b303","eni-03491dab1d5e3ce08"
    ]
    account = "298186006663"
  }
}
#
resource "confluent_kafka_cluster" "enterprise" {
  display_name = "enterprise_kafka_cluster"
  availability = "HIGH"
  cloud        = "AWS"
  region       = "us-west-2"
  enterprise {}

  environment {
    id = confluent_environment.development.id
  }
}
#
variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also referred as Cloud API ID)"
  type        = string
  default     = "X4Q5RLKW6V7T6VTJ"
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret"
  type        = string
  sensitive   = true
  default     = "WwveMWbHu2XalIjUJD72WwWnVHVjsgGr3yM0ehv/esiyKhdK0BpmVooSyV03tNRU"
}
