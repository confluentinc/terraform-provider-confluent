terraform {
  required_providers {
    confluentcloud = {
      source  = "confluentinc/confluentcloud"
      version = "0.6.0"
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

# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluentcloud/latest/docs/resources/confluentcloud_kafka_cluster
resource "confluentcloud_kafka_cluster" "standard" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-2"
  standard {}
  environment {
    id = confluentcloud_environment.staging.id
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
  crn_pattern = confluentcloud_kafka_cluster.standard.rbac_crn
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
    id          = confluentcloud_kafka_cluster.standard.id
    api_version = confluentcloud_kafka_cluster.standard.api_version
    kind        = confluentcloud_kafka_cluster.standard.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }

  # The goal is to ensure that confluentcloud_role_binding.app-manager-kafka-cluster-admin is created before
  # confluentcloud_api_key.app-manager-kafka-api-key is used to create instances of
  # confluentcloud_kafka_topic, confluentcloud_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluentcloud_api_key.app-manager-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluentcloud_kafka_topic, confluentcloud_kafka_acl resources instead.
  depends_on = [
    confluentcloud_role_binding.app-manager-kafka-cluster-admin
  ]
}

resource "confluentcloud_kafka_topic" "orders" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.standard.id
  }
  topic_name    = "orders"
  http_endpoint = confluentcloud_kafka_cluster.standard.http_endpoint
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
    id          = confluentcloud_kafka_cluster.standard.id
    api_version = confluentcloud_kafka_cluster.standard.api_version
    kind        = confluentcloud_kafka_cluster.standard.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }
}

resource "confluentcloud_kafka_acl" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.standard.id
  }
  resource_type = "TOPIC"
  resource_name = confluentcloud_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluentcloud_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.standard.http_endpoint
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
    id          = confluentcloud_kafka_cluster.standard.id
    api_version = confluentcloud_kafka_cluster.standard.api_version
    kind        = confluentcloud_kafka_cluster.standard.kind

    environment {
      id = confluentcloud_environment.staging.id
    }
  }
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluentcloud_kafka_acl.app-consumer-read-on-topic, confluentcloud_kafka_acl.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluentcloud_kafka_acl" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.standard.id
  }
  resource_type = "TOPIC"
  resource_name = confluentcloud_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluentcloud_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.standard.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluentcloud_kafka_acl" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.standard.id
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
  http_endpoint = confluentcloud_kafka_cluster.standard.http_endpoint
  credentials {
    key    = confluentcloud_api_key.app-manager-kafka-api-key.id
    secret = confluentcloud_api_key.app-manager-kafka-api-key.secret
  }
}
