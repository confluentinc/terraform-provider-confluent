terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.11.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "4.18.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment_v2" "staging" {
  display_name = "Staging"
}

resource "confluent_network_v1" "peering" {
  display_name     = "Peering Network"
  cloud            = "GCP"
  region           = var.region
  cidr             = var.cidr
  connection_types = ["PEERING"]
  environment {
    id = confluent_environment_v2.staging.id
  }
}

resource "confluent_peering_v1" "gcp" {
  display_name = "GCP Peering"
  gcp {
    project              = var.customer_project_id
    vpc_network          = var.customer_vpc_network
    import_custom_routes = var.import_custom_routes
  }
  environment {
    id = confluent_environment_v2.staging.id
  }
  network {
    id = confluent_network_v1.peering.id
  }
}

resource "confluent_kafka_cluster_v2" "dedicated" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = confluent_network_v1.peering.cloud
  region       = confluent_network_v1.peering.region
  dedicated {
    cku = 1
  }
  environment {
    id = confluent_environment_v2.staging.id
  }
  network {
    id = confluent_network_v1.peering.id
  }
}

// 'app-manager' service account is required in this configuration to create 'orders' topic and grant ACLs
// to 'app-producer' and 'app-consumer' service accounts.
resource "confluent_service_account_v2" "app-manager" {
  display_name = "app-manager"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding_v2" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account_v2.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster_v2.dedicated.rbac_crn
}

resource "confluent_api_key_v2" "app-manager-kafka-api-key" {
  display_name = "app-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account_v2.app-manager.id
    api_version = confluent_service_account_v2.app-manager.api_version
    kind        = confluent_service_account_v2.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.dedicated.id
    api_version = confluent_kafka_cluster_v2.dedicated.api_version
    kind        = confluent_kafka_cluster_v2.dedicated.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }

  # The goal is to ensure that
  # 1. confluent_role_binding_v2.app-manager-kafka-cluster-admin is created before
  # confluent_api_key_v2.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic_v3, confluent_kafka_acl_v3 resources.
  # 2. Kafka connectivity through GCP VPC Peering is setup.
  depends_on = [
    confluent_role_binding_v2.app-manager-kafka-cluster-admin,

    confluent_peering_v1.gcp,
    google_compute_network_peering.peering
  ]
}

resource "confluent_kafka_topic_v3" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.dedicated.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster_v2.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account_v2" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key_v2" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"
  owner {
    id          = confluent_service_account_v2.app-consumer.id
    api_version = confluent_service_account_v2.app-consumer.api_version
    kind        = confluent_service_account_v2.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.dedicated.id
    api_version = confluent_kafka_cluster_v2.dedicated.api_version
    kind        = confluent_kafka_cluster_v2.dedicated.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }

  depends_on = [
    confluent_peering_v1.gcp,
    google_compute_network_peering.peering
  ]
}

resource "confluent_kafka_acl_v3" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic_v3.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account_v2.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account_v2" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key_v2" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"
  owner {
    id          = confluent_service_account_v2.app-producer.id
    api_version = confluent_service_account_v2.app-producer.api_version
    kind        = confluent_service_account_v2.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.dedicated.id
    api_version = confluent_kafka_cluster_v2.dedicated.api_version
    kind        = confluent_kafka_cluster_v2.dedicated.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }

  depends_on = [
    confluent_peering_v1.gcp,
    google_compute_network_peering.peering
  ]
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluent_kafka_acl_v3.app-consumer-read-on-topic, confluent_kafka_acl_v3.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluent_kafka_acl_v3" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic_v3.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account_v2.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl_v3" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.dedicated.id
  }
  resource_type = "GROUP"
  // The existing values of resource_name, pattern_type attributes are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_v3_consume.html
  // Update the values of resource_name, pattern_type attributes to match your target consumer group ID.
  // https://docs.confluent.io/platform/current/kafka/authorization.html#prefixed-acls
  resource_name = "confluent_cli_consumer_"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account_v2.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
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
  peer_network = "projects/${confluent_network_v1.peering.gcp[0].project}/global/networks/${confluent_network_v1.peering.gcp[0].vpc_network}"
}
