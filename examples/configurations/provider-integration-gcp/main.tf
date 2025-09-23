terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "~> 1.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

# Step 1: Create the provider integration (DRAFT status)
resource "confluent_provider_integration_setup" "gcp" {
  environment {
    id = var.environment_id
  }
  
  display_name   = var.integration_display_name
  cloud_provider = "gcp"
}

# Step 2: Configure and validate GCP integration
resource "confluent_provider_integration_authorization" "gcp" {
  provider_integration_id = confluent_provider_integration_setup.gcp.id
  
  environment {
    id = var.environment_id
  }
  
  gcp {
    customer_google_service_account = var.gcp_service_account
  }
}