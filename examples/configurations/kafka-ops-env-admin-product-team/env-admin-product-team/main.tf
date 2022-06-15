terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.11.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

data "confluent_environment_v2" "staging" {
  id = var.environment_id
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
    id = data.confluent_environment_v2.staging.id
  }
}

data "confluent_service_account_v2" "env-manager" {
  id = var.env_manager_id
}

resource "confluent_api_key_v2" "env-manager-kafka-api-key" {
  display_name = "env-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'env-manager' service account"
  owner {
    id          = data.confluent_service_account_v2.env-manager.id
    api_version = data.confluent_service_account_v2.env-manager.api_version
    kind        = data.confluent_service_account_v2.env-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = data.confluent_environment_v2.staging.id
    }
  }
}

resource "confluent_kafka_topic_v3" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster_v2.basic.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster_v2.basic.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.env-manager-kafka-api-key.id
    secret = confluent_api_key_v2.env-manager-kafka-api-key.secret
  }
}

data "confluent_service_account_v2" "app-consumer" {
  id = var.app_consumer_id
}

resource "confluent_api_key_v2" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"
  owner {
    id          = data.confluent_service_account_v2.app-consumer.id
    api_version = data.confluent_service_account_v2.app-consumer.api_version
    kind        = data.confluent_service_account_v2.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = data.confluent_environment_v2.staging.id
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
  principal     = "User:${data.confluent_service_account_v2.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.basic.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.env-manager-kafka-api-key.id
    secret = confluent_api_key_v2.env-manager-kafka-api-key.secret
  }
}


data "confluent_service_account_v2" "app-producer" {
  id = var.app_producer_id
}

resource "confluent_api_key_v2" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"
  owner {
    id          = data.confluent_service_account_v2.app-producer.id
    api_version = data.confluent_service_account_v2.app-producer.api_version
    kind        = data.confluent_service_account_v2.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster_v2.basic.id
    api_version = confluent_kafka_cluster_v2.basic.api_version
    kind        = confluent_kafka_cluster_v2.basic.kind

    environment {
      id = data.confluent_environment_v2.staging.id
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
  principal     = "User:${data.confluent_service_account_v2.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.basic.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.env-manager-kafka-api-key.id
    secret = confluent_api_key_v2.env-manager-kafka-api-key.secret
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
  principal     = "User:${data.confluent_service_account_v2.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster_v2.basic.rest_endpoint
  credentials {
    key    = confluent_api_key_v2.env-manager-kafka-api-key.id
    secret = confluent_api_key_v2.env-manager-kafka-api-key.secret
  }
}
