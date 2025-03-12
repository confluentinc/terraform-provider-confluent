terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.20.0"
    }

    google = {
        source  = "hashicorp/google"
        version = "6.21.0"
    }
  }
}


provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

provider "google" {
  features {
  }
  project = var.project_id
  region = var.region
}

resource "confluent_private_link_attachment" "main" {
  cloud        = "GCP"
  region       = var.region
  display_name = "staging-gcp-platt"
  environment {
    id = var.confluent_cloud_environment_id
  }
}

resource "confluent_private_link_attachment_connection" "main" {
  display_name = "staging-gcp-plattc"
  environment {
    id = var.confluent_cloud_environment_id
  }
  gcp {
    private_service_connect_connection_id = var.connection_id
  }

  private_link_attachment {
    id = confluent_private_link_attachment.main.id
  }
}
