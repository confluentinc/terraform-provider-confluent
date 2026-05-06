terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.72.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "main" {
  display_name = var.environment_name
}

resource "confluent_gateway" "gcp_ingress" {
  display_name = "gcp-ingress-gateway"
  environment {
    id = confluent_environment.main.id
  }
  gcp_ingress_private_service_connect_gateway {
    region = var.region
  }
}

resource "confluent_kafka_cluster" "enterprise" {
  display_name = "${var.resource_prefix}-cluster"
  availability = "HIGH"
  cloud        = "GCP"
  region       = var.region
  enterprise {}
  environment {
    id = confluent_environment.main.id
  }
}

resource "confluent_access_point" "gcp_ingress" {
  display_name = "${var.resource_prefix}-gcp-ingress-ap"
  environment {
    id = confluent_environment.main.id
  }
  gateway {
    id = confluent_gateway.gcp_ingress.id
  }
  gcp_ingress_private_service_connect_endpoint {
    private_service_connect_connection_id = var.private_service_connect_connection_id
  }
}

resource "confluent_service_account" "app-manager" {
  display_name = "${var.resource_prefix}-app-manager"
  description  = "Service account to manage Kafka cluster"
}

resource "confluent_role_binding" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.enterprise.rbac_crn
}

resource "confluent_api_key" "app-manager-kafka-api-key" {
  display_name = "${var.resource_prefix}-app-manager-kafka-api-key"
  description  = "Kafka API Key owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.enterprise.id
    api_version = confluent_kafka_cluster.enterprise.api_version
    kind        = confluent_kafka_cluster.enterprise.kind

    environment {
      id = confluent_environment.main.id
    }
  }

  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin,
    confluent_access_point.gcp_ingress
  ]
}
