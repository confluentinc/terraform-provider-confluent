terraform {
  required_version = ">= 0.14.0"
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.36.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.17.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "staging" {
  display_name = "Staging"

  stream_governance {
    package = "ESSENTIALS"
  }
}

data "confluent_schema_registry_cluster" "essentials" {
  environment {
    id = confluent_environment.staging.id
  }

  depends_on = [
    confluent_kafka_cluster.dedicated
  ]
}

resource "confluent_network" "transit-gateway" {
  display_name     = "Transit Gateway Network"
  cloud            = "AWS"
  region           = var.region
  cidr             = var.cidr
  connection_types = ["TRANSITGATEWAY"]
  environment {
    id = confluent_environment.staging.id
  }
}

resource "confluent_transit_gateway_attachment" "aws" {
  display_name = "AWS Transit Gateway Attachment"
  aws {
    ram_resource_share_arn = aws_ram_resource_share.confluent.arn
    transit_gateway_id     = data.aws_ec2_transit_gateway.input.id
    routes                 = var.routes
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.transit-gateway.id
  }
}

resource "confluent_kafka_cluster" "dedicated" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = confluent_network.transit-gateway.cloud
  region       = confluent_network.transit-gateway.region
  dedicated {
    cku = 1
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.transit-gateway.id
  }
}

// 'app-manager' service account is required in this configuration to create 'orders' topic and grant ACLs
// to 'app-producer' and 'app-consumer' service accounts.
resource "confluent_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.dedicated.rbac_crn
}

resource "confluent_api_key" "app-manager-kafka-api-key" {
  display_name = "app-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager' service account"

  # Set optional `disable_wait_for_ready` attribute (defaults to `false`) to `true` if the machine where Terraform is not run within a private network
  # disable_wait_for_ready = true

  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that
  # 1. confluent_role_binding.app-manager-kafka-cluster-admin is created before
  # confluent_api_key.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.
  # 2. Kafka connectivity through AWS Transit Gateway is setup.
  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin,

    confluent_transit_gateway_attachment.aws,
    aws_route.r
  ]
}

// Provisioning Kafka Topics requires access to the REST endpoint on the Kafka cluster
// If Terraform is not run from within the private network, this will not work
resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"

  # Set optional `disable_wait_for_ready` attribute (defaults to `false`) to `true` if the machine where Terraform is not run within a private network
  # disable_wait_for_ready = true

  owner {
    id          = confluent_service_account.app-consumer.id
    api_version = confluent_service_account.app-consumer.api_version
    kind        = confluent_service_account.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  depends_on = [
    confluent_transit_gateway_attachment.aws,
    aws_route.r
  ]
}

// Provisioning Kafka ACLs requires access to the REST endpoint on the Kafka cluster
// If Terraform is not run from within the private network, this will not work
resource "confluent_kafka_acl" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"

  # Set optional `disable_wait_for_ready` attribute (defaults to `false`) to `true` if the machine where Terraform is not run within a private network
  # disable_wait_for_ready = true

  owner {
    id          = confluent_service_account.app-producer.id
    api_version = confluent_service_account.app-producer.api_version
    kind        = confluent_service_account.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  depends_on = [
    confluent_transit_gateway_attachment.aws,
    aws_route.r
  ]
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluent_kafka_acl.app-consumer-read-on-topic, confluent_kafka_acl.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls

// Provisioning Kafka ACLs requires access to the REST endpoint on the Kafka cluster
// If Terraform is not run from within the private network, this will not work
resource "confluent_kafka_acl" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

// Provisioning Kafka ACLs requires access to the REST endpoint on the Kafka cluster
// If Terraform is not run from within the private network, this will not work
resource "confluent_kafka_acl" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "GROUP"
  // The existing values of resource_name, pattern_type attributes are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_consume.html
  // Update the values of resource_name, pattern_type attributes to match your target consumer group ID.
  // https://docs.confluent.io/platform/current/kafka/authorization.html#prefixed-acls
  resource_name = "confluent_cli_consumer_"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

# https://docs.confluent.io/cloud/current/networking/aws-transit-gateway.html
# Create a Transit Gateway Connection to Confluent Cloud on AWS
provider "aws" {
  region = var.customer_region
}

# Sharing Transit Gateway with Confluent via Resource Access Manager (RAM) Resource Share
resource "aws_ram_resource_share" "confluent" {
  name                      = "resource-share-with-confluent"
  allow_external_principals = true
}

resource "aws_ram_principal_association" "confluent" {
  principal          = confluent_network.transit-gateway.aws[0].account
  resource_share_arn = aws_ram_resource_share.confluent.arn
}

data "aws_ec2_transit_gateway" "input" {
  id = var.transit_gateway_id
}

resource "aws_ram_resource_association" "example" {
  resource_arn       = data.aws_ec2_transit_gateway.input.arn
  resource_share_arn = aws_ram_resource_share.confluent.arn
}

# Accepter's side of the connection.
data "aws_ec2_transit_gateway_vpc_attachment" "accepter" {
  id = confluent_transit_gateway_attachment.aws.aws[0].transit_gateway_attachment_id
}

# Accept Transit Gateway Attachment from Confluent
resource "aws_ec2_transit_gateway_vpc_attachment_accepter" "accepter" {
  transit_gateway_attachment_id = data.aws_ec2_transit_gateway_vpc_attachment.accepter.id
}

data "aws_subnets" "input" {
  filter {
    name   = "vpc-id"
    values = [var.vpc_id]
  }
}

# Create Transit Gateway Attachment for the user's VPC
resource "aws_ec2_transit_gateway_vpc_attachment" "attachment" {
  subnet_ids         = data.aws_subnets.input.ids
  transit_gateway_id = data.aws_ec2_transit_gateway.input.id
  vpc_id             = var.vpc_id
}

# Find the routing table
data "aws_route_tables" "rts" {
  vpc_id = var.vpc_id
}

resource "aws_route" "r" {
  for_each               = toset(data.aws_route_tables.rts.ids)
  route_table_id         = each.key
  destination_cidr_block = confluent_network.transit-gateway.cidr
  transit_gateway_id     = data.aws_ec2_transit_gateway.input.id
}
