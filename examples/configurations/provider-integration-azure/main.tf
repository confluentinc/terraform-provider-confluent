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
resource "confluent_provider_integration_v2" "azure" {
  environment {
    id = var.environment_id
  }
  
  display_name   = var.integration_display_name
  cloud_provider = "azure"
}

# Step 2: Configure and validate Azure integration
resource "confluent_provider_integration_v2_authorization" "azure" {
  provider_integration_id = confluent_provider_integration_v2.azure.id
  
  environment {
    id = var.environment_id
  }
  
  azure {
    customer_azure_tenant_id = var.azure_tenant_id
  }
}