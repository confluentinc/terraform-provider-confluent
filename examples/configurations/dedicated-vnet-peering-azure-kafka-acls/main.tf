terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluentcloud = {
      source  = "confluentcloud"
      version = "0.6.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 2.55.0"
    }
    azuread = {
      source  = "hashicorp/azuread"
      version = "~> 2.15.0"
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
  cloud            = "AZURE"
  region           = var.region
  cidr             = var.cidr
  connection_types = ["PEERING"]
  environment {
    id = confluentcloud_environment.staging.id
  }
}

resource "confluentcloud_peering" "azure" {
  display_name = "Azure Peering"
  azure {
    tenant          = var.tenant_id
    vnet            = local.vnet_resource_id
    customer_region = var.customer_region
  }
  environment {
    id = confluentcloud_environment.staging.id
  }
  network {
    id = confluentcloud_network.peering.id
  }

  depends_on = [
    time_sleep.wait_30_seconds_after_peering_creator_role_assignment_creation
  ]
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

// 'app-manager' service account is required in this configuration to create 'orders' topic and grant ACLs
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
  # confluentcloud_kafka_topic, confluentcloud_kafka_acl resources.
  # 2. Kafka connectivity through Azure VNet Peering is setup.
  depends_on = [
    confluentcloud_role_binding.app-manager-kafka-cluster-admin,

    confluentcloud_peering.azure,
    time_sleep.wait_30_seconds_after_peering_creator_role_assignment_creation
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
    confluentcloud_peering.azure,
    time_sleep.wait_30_seconds_after_peering_creator_role_assignment_creation
  ]
}

resource "confluentcloud_kafka_acl" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluentcloud_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluentcloud_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.dedicated.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
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
    confluentcloud_peering.azure,
    time_sleep.wait_30_seconds_after_peering_creator_role_assignment_creation
  ]
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluentcloud_kafka_acl.app-consumer-read-on-topic, confluentcloud_kafka_acl.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluentcloud_kafka_acl" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluentcloud_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluentcloud_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.dedicated.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluentcloud_kafka_acl" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.dedicated.id
  }
  resource_type = "GROUP"
  // The existing values of resource_name, pattern_type attributes are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_consume.html
  // Update the values of resource_name, pattern_type attributes to match your target consumer group ID.
  // https://docs.confluent.io/platform/current/kafka/authorization.html#prefixed-acls
  resource_name = "confluent_cli_consumer_"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluentcloud_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.dedicated.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
}

# https://docs.confluent.io/cloud/current/networking/peering/azure-peering.html#create-a-vnet-peering-connection-to-ccloud-on-az
# Create a VNet Peering Connection to Confluent Cloud on Azure
provider "azurerm" {
  features {
  }
  subscription_id = var.subscription_id
  client_id       = var.client_id
  client_secret   = var.client_secret
  tenant_id       = var.tenant_id
}

resource "azurerm_role_definition" "peering_creator" {
  name        = "Confluent Cloud Peering Creator"
  scope       = "/subscriptions/${var.subscription_id}"
  description = "Perform cross-tenant network peering."

  permissions {
    actions = [
      "Microsoft.Network/virtualNetworks/read",
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/read",
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/write",
      "Microsoft.Network/virtualNetworks/virtualNetworkPeerings/delete",
      "Microsoft.Network/virtualNetworks/peer/action"
    ]
    not_actions = []
  }

  assignable_scopes = [
    "/subscriptions/${var.subscription_id}",
  ]
}

locals {
  vnet_resource_id = "/subscriptions/${var.subscription_id}/resourceGroups/${var.resource_group_name}/providers/Microsoft.Network/virtualNetworks/${var.vnet_name}"
}

# Configure the Azure Active Directory Provider
provider "azuread" {
  client_id     = var.client_id
  client_secret = var.client_secret
  tenant_id     = var.tenant_id
}

data "azuread_service_principal" "peering_creator" {
  # Harcoded Confluent's client_id
  application_id = "f0955e3a-9013-4cf4-a1ea-21587621c9cc"
}

resource "azurerm_role_assignment" "peering_creator" {
  scope              = local.vnet_resource_id
  role_definition_id = azurerm_role_definition.peering_creator.role_definition_resource_id
  principal_id       = data.azuread_service_principal.peering_creator.object_id
}

resource "null_resource" "previous" {}
resource "time_sleep" "wait_30_seconds_after_peering_creator_role_assignment_creation" {
  depends_on = [null_resource.previous]

  create_duration = "30s"
}