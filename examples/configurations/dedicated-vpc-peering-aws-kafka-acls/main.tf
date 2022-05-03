terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluentcloud = {
      source  = "confluentinc/confluentcloud"
      version = "0.6.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "= 2.32.0"
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
  cloud            = "AWS"
  region           = var.region
  cidr             = var.cidr
  connection_types = ["PEERING"]
  environment {
    id = confluentcloud_environment.staging.id
  }
}

resource "confluentcloud_peering" "aws" {
  display_name = "AWS Peering"
  aws {
    account         = var.aws_account_id
    vpc             = var.vpc_id
    routes          = var.routes
    customer_region = var.customer_region
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
  # 2. Kafka connectivity through AWS VPC Peering is setup.
  depends_on = [
    confluentcloud_role_binding.app-manager-kafka-cluster-admin,

    confluentcloud_peering.aws,
    aws_route.r
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
    confluentcloud_peering.aws,
    aws_route.r
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
    confluentcloud_peering.aws,
    aws_route.r
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

# https://docs.confluent.io/cloud/current/networking/peering/aws-peering.html
# Create a VPC Peering Connection to Confluent Cloud on AWS
provider "aws" {
  region = var.customer_region
}

# Accepter's side of the connection.
# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/vpc_peering_connection
data "aws_vpc_peering_connection" "accepter" {
  vpc_id      = confluentcloud_network.peering.aws[0].vpc
  peer_vpc_id = confluentcloud_peering.aws.aws[0].vpc
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/vpc_peering_connection_accepter
resource "aws_vpc_peering_connection_accepter" "peer" {
  vpc_peering_connection_id = data.aws_vpc_peering_connection.accepter.id
  auto_accept               = true
}

# Find the routing table
data "aws_route_tables" "rts" {
  vpc_id = confluentcloud_peering.aws.aws[0].vpc
}

resource "aws_route" "r" {
  for_each                  = toset(data.aws_route_tables.rts.ids)
  route_table_id            = each.key
  destination_cidr_block    = confluentcloud_network.peering.cidr
  vpc_peering_connection_id = data.aws_vpc_peering_connection.accepter.id

  depends_on = [
    aws_vpc_peering_connection_accepter.peer
  ]
}
