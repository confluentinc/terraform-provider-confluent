terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluentcloud = {
      source  = "confluentinc/confluentcloud"
      version = "0.6.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "4.18.0"
    }
  }
}

provider "confluentcloud" {
  api_key    = var.confluent_cloud_api_key
  api_secret = var.confluent_cloud_api_secret
}

resource "confluentcloud_environment" "staging" {
  display_name = "Staging"
}

resource "confluentcloud_network" "peering" {
  display_name     = "Peering Network"
  cloud            = "GCP"
  region           = var.region
  cidr             = var.cidr
  connection_types = ["PEERING"]
  environment {
    id = confluentcloud_environment.staging.id
  }
}

resource "confluentcloud_peering" "gcp" {
  display_name = "GCP Peering"
  gcp {
    project       = var.customer_project_id
    vpc_network   = var.customer_vpc_network
    custom_routes = var.custom_routes
  }
  environment {
    id = confluentcloud_environment.staging.id
  }
  network {
    id = confluentcloud_network.peering.id
  }
}

resource "confluentcloud_kafka_cluster" "dedicated" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = confluentcloud_network.peering.cloud
  region       = confluentcloud_network.peering.region
  dedicated {
    cku = 1
  }
  environment {
    id = confluentcloud_environment.staging.id
  }
  network {
    id = confluentcloud_network.peering.id
  }
}

// 'app-manager' service account is required in this configuration to create 'orders' topic and assign roles
// to 'app-producer' and 'app-consumer' service accounts.
resource "confluentcloud_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluentcloud_role_binding" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluentcloud_service_account.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluentcloud_kafka_cluster.dedicated.rbac_crn
}

resource "confluentcloud_api_key" "app-manager-kafka-api-key" {
  display_name = "app-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluentcloud_service_account.app-manager.id
    api_version = confluentcloud_service_account.app-manager.api_version
    kind        = confluentcloud_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluentcloud_kafka_cluster.dedicated.id
    api_version = confluentcloud_kafka_cluster.dedicated.api_version
    kind        = confluentcloud_kafka_cluster.dedicated.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }

  # The goal is to ensure that
  # 1. confluentcloud_role_binding.app-manager-kafka-cluster-admin is created before
  # confluentcloud_api_key.app-manager-kafka-api-key is used to create instances of
  # confluentcloud_kafka_topic resource.
  # 2. Kafka connectivity through GCP VPC Peering is setup.
  depends_on = [
    confluentcloud_role_binding.app-manager-kafka-cluster-admin,

    confluentcloud_peering.gcp,
    google_compute_network_peering.peering
  ]
}

resource "confluentcloud_kafka_topic" "orders" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.dedicated.id
  }
  topic_name    = "orders"
  http_endpoint = confluentcloud_kafka_cluster.dedicated.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluentcloud_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluentcloud_api_key" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"
  owner {
    id          = confluentcloud_service_account.app-consumer.id
    api_version = confluentcloud_service_account.app-consumer.api_version
    kind        = confluentcloud_service_account.app-consumer.kind
  }

  managed_resource {
    id          = confluentcloud_kafka_cluster.dedicated.id
    api_version = confluentcloud_kafka_cluster.dedicated.api_version
    kind        = confluentcloud_kafka_cluster.dedicated.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }

  depends_on = [
    confluentcloud_peering.gcp,
    google_compute_network_peering.peering
  ]
}

resource "confluentcloud_role_binding" "app-producer-developer-write" {
  principal   = "User:${confluentcloud_service_account.app-producer.id}"
  role_name   = "DeveloperWrite"
  crn_pattern = "${confluentcloud_kafka_cluster.dedicated.rbac_crn}/kafka=${confluentcloud_kafka_cluster.dedicated.id}/topic=${confluentcloud_kafka_topic.orders.topic_name}"
}

resource "confluentcloud_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluentcloud_api_key" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"
  owner {
    id          = confluentcloud_service_account.app-producer.id
    api_version = confluentcloud_service_account.app-producer.api_version
    kind        = confluentcloud_service_account.app-producer.kind
  }

  managed_resource {
    id          = confluentcloud_kafka_cluster.dedicated.id
    api_version = confluentcloud_kafka_cluster.dedicated.api_version
    kind        = confluentcloud_kafka_cluster.dedicated.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }

  depends_on = [
    confluentcloud_peering.gcp,
    google_compute_network_peering.peering
  ]
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
resource "confluentcloud_role_binding" "app-producer-developer-read-from-topic" {
  principal   = "User:${confluentcloud_service_account.app-consumer.id}"
  role_name   = "DeveloperRead"
  crn_pattern = "${confluentcloud_kafka_cluster.dedicated.rbac_crn}/kafka=${confluentcloud_kafka_cluster.dedicated.id}/topic=${confluentcloud_kafka_topic.orders.topic_name}"
}

resource "confluentcloud_role_binding" "app-producer-developer-read-from-group" {
  principal = "User:${confluentcloud_service_account.app-consumer.id}"
  role_name = "DeveloperRead"
  // The existing value of crn_pattern's suffix (group=confluent_cli_consumer_*) are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_consume.html
  // Update it to match your target consumer group ID.
  crn_pattern = "${confluentcloud_kafka_cluster.dedicated.rbac_crn}/kafka=${confluentcloud_kafka_cluster.dedicated.id}/group=confluent_cli_consumer_*"
}

# Set GOOGLE_APPLICATION_CREDENTIALS environment variable to a path to a key file
# for Google TF Provider to work: https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#adding-credentials
provider "google" {
  project = var.customer_project_id
  region  = var.customer_region
}

# https://docs.confluent.io/cloud/current/networking/peering/gcp-peering.html
# Create a VPC Peering Connection to Confluent Cloud on Google Cloud
resource "google_compute_network_peering" "peering" {
  name         = var.customer_peering_name
  network      = "projects/${var.customer_project_id}/global/networks/${var.customer_vpc_network}"
  peer_network = "projects/${confluentcloud_network.peering.gcp[0].project}/global/networks/${confluentcloud_network.peering.gcp[0].vpc_network}"
}
