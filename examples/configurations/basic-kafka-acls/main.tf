terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.10.0"
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

# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster_v2
resource "confluent_kafka_cluster_v2" "basic" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-2"
  basic {}
  environment {
    id = confluent_environment_v2.staging.id
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
  crn_pattern = confluent_kafka_cluster_v2.basic.rbac_crn
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
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }

  # The goal is to ensure that confluent_role_binding_v2.app-manager-kafka-cluster-admin is created before
  # confluent_api_key_v2.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic_v3, confluent_kafka_acl_v3 resources.

  # 'depends_on' meta-argument is specified in confluent_api_key_v2.app-manager-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic_v3, confluent_kafka_acl_v3 resources instead.
  depends_on = [
    confluent_role_binding_v2.app-manager-kafka-cluster-admin
  ]
}

resource "confluent_kafka_topic_v3" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.basic.id
  }
  topic_name    = "orders"
  http_endpoint = confluent_kafka_cluster_v2.basic.http_endpoint
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
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }
}

resource "confluent_kafka_acl_v3" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.basic.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic_v3.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account_v2.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  http_endpoint = confluent_kafka_cluster_v2.basic.http_endpoint
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
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = confluent_environment_v2.staging.id
    }
  }
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluent_kafka_acl_v3.app-consumer-read-on-topic, confluent_kafka_acl_v3.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluent_kafka_acl_v3" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.basic.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic_v3.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account_v2.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  http_endpoint = confluent_kafka_cluster_v2.basic.http_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl_v3" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.basic.id
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
  http_endpoint = confluent_kafka_cluster_v2.basic.http_endpoint
  credentials {
    key    = confluent_api_key_v2.app-manager-kafka-api-key.id
    secret = confluent_api_key_v2.app-manager-kafka-api-key.secret
  }
}
