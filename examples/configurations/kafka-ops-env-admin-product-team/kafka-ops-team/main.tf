terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.30.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "staging" {
  display_name = "Staging"
}

resource "confluent_schema_registry_cluster" "essentials" {
  package = "ESSENTIALS"

  environment {
    id = confluent_environment.staging.id
  }

  region {
    # See https://docs.confluent.io/cloud/current/stream-governance/packages.html#stream-governance-regions
    id = "sgreg-1"
  }
}

resource "confluent_service_account" "env-manager" {
  display_name = "env-manager"
  description  = "Service account to manage resources under 'Staging' environment"
}

resource "confluent_role_binding" "env-manager-env-admin" {
  principal   = "User:${confluent_service_account.env-manager.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = confluent_environment.staging.resource_name
}

resource "confluent_api_key" "env-manager-cloud-api-key" {
  display_name = "env-manager-cloud-api-key"
  description  = "Cloud API Key to be shared with Product team to manage resources under 'Staging' environment"
  owner {
    id          = confluent_service_account.env-manager.id
    api_version = confluent_service_account.env-manager.api_version
    kind        = confluent_service_account.env-manager.kind
  }

  depends_on = [
    confluent_role_binding.env-manager-env-admin
  ]
}

# Kafka Ops team creates all service accounts for now until CIAM-1882 is resolved since OrganizationAdmin role is required for SA creation
resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}
