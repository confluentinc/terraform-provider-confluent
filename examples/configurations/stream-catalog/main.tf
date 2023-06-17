terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.45.0"
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

# Stream Governance and Kafka clusters can be in different regions as well as different cloud providers,
# but you should to place both in the same cloud and region to restrict the fault isolation boundary.
data "confluent_schema_registry_region" "main" {
  cloud   = "AWS"
  region  = "us-east-2"
  package = "ADVANCED"
}

resource "confluent_schema_registry_cluster" "main" {
  package = data.confluent_schema_registry_region.main.package

  environment {
    id = confluent_environment.staging.id
  }

  region {
    # See https://docs.confluent.io/cloud/current/stream-governance/packages.html#stream-governance-regions
    id = data.confluent_schema_registry_region.main.id
  }
}

# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster
resource "confluent_kafka_cluster" "basic" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-2"
  basic {}
  environment {
    id = confluent_environment.staging.id
  }
}

// 'app-manager' service account is required in this configuration to create 'purchase' topic and grant ACLs
// to 'app-producer' and 'app-consumer' service accounts.
resource "confluent_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.basic.rbac_crn
}

resource "confluent_api_key" "app-manager-kafka-api-key" {
  display_name = "app-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.basic.id
    api_version = confluent_kafka_cluster.basic.api_version
    kind        = confluent_kafka_cluster.basic.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager-kafka-cluster-admin is created before
  # confluent_api_key.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin
  ]
}

resource "confluent_kafka_topic" "purchase" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  topic_name    = "purchase"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'purchase' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"
  owner {
    id          = confluent_service_account.app-consumer.id
    api_version = confluent_service_account.app-consumer.api_version
    kind        = confluent_service_account.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.basic.id
    api_version = confluent_kafka_cluster.basic.api_version
    kind        = confluent_kafka_cluster.basic.kind

    environment {
      id = confluent_environment.staging.id
    }
  }
}

resource "confluent_kafka_acl" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.purchase.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'purchase' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"
  owner {
    id          = confluent_service_account.app-producer.id
    api_version = confluent_service_account.app-producer.api_version
    kind        = confluent_service_account.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.basic.id
    api_version = confluent_kafka_cluster.basic.api_version
    kind        = confluent_kafka_cluster.basic.kind

    environment {
      id = confluent_environment.staging.id
    }
  }
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluent_kafka_acl.app-consumer-read-on-topic, confluent_kafka_acl.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluent_kafka_acl" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.purchase.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
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
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "env-manager" {
  display_name = "env-manager"
  description  = "Service account to manage 'Staging' environment"
}

resource "confluent_role_binding" "env-manager-data-steward" {
  principal   = "User:${confluent_service_account.env-manager.id}"
  role_name   = "DataSteward"
  crn_pattern = confluent_environment.staging.resource_name
}

resource "confluent_api_key" "env-manager-schema-registry-api-key" {
  display_name = "env-manager-schema-registry-api-key"
  description  = "Schema Registry API Key that is owned by 'env-manager' service account"
  owner {
    id          = confluent_service_account.env-manager.id
    api_version = confluent_service_account.env-manager.api_version
    kind        = confluent_service_account.env-manager.kind
  }

  managed_resource {
    id          = confluent_schema_registry_cluster.main.id
    api_version = confluent_schema_registry_cluster.main.api_version
    kind        = confluent_schema_registry_cluster.main.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that confluent_role_binding.env-manager-data-steward is created before
  # confluent_api_key.env-manager-schema-registry-api-key is used to create instances of
  # confluent_schema resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.env-manager-schema-registry-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_schema resources instead.
  depends_on = [
    confluent_role_binding.env-manager-data-steward
  ]
}

resource "confluent_schema" "purchase" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.purchase.topic_name}-value"
  format = "AVRO"
  schema = file("./schemas/avro/purchase.avsc")
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }
}

resource "confluent_tag" "pii" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  name = "PII"
  description = "Personally identifiable information"
}

resource "confluent_tag" "sensitive" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  name = "Sensitive"
  description = "Sensitive tag description"
}

resource "confluent_tag" "private" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  name = "Private"
  description = "Private tag description"
}

resource "confluent_business_metadata" "main" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  name = "Domain"
  description = "These are events that describe the domain of activity."
  attribute_definition {
    name = "Team_owner"
  }
  attribute_definition {
    name = "Slack_contact"
  }
}

# Apply the Tag/BusinessMetadata on a topic
resource "confluent_tag_binding" "pii-topic-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.pii.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:${confluent_kafka_cluster.basic.id}:${confluent_kafka_topic.purchase.topic_name}"
  entity_type = local.topic_entity_type
}

resource "confluent_tag_binding" "sensitive-topic-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.sensitive.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:${confluent_kafka_cluster.basic.id}:${confluent_kafka_topic.purchase.topic_name}"
  entity_type = local.topic_entity_type
}

resource "confluent_tag_binding" "private-topic-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.private.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:${confluent_kafka_cluster.basic.id}:${confluent_kafka_topic.purchase.topic_name}"
  entity_type = local.topic_entity_type
}

resource "confluent_business_metadata_binding" "main" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  business_metadata_name = confluent_business_metadata.main.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:${confluent_kafka_cluster.basic.id}:${confluent_kafka_topic.purchase.topic_name}"
  entity_type = local.topic_entity_type
  attributes = {
    "Team_owner" = "Sam"
    "Slack_contact" = "@sam"
  }
}

# Apply the Tag/BusinessMetadata on a schema
resource "confluent_tag_binding" "pii-schema-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.pii.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:.:${confluent_schema.purchase.schema_identifier}"
  entity_type = local.schema_entity_type
}

resource "confluent_tag_binding" "sensitive-schema-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.sensitive.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:.:${confluent_schema.purchase.schema_identifier}"
  entity_type = local.schema_entity_type
}

resource "confluent_tag_binding" "private-schema-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.private.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:.:${confluent_schema.purchase.schema_identifier}"
  entity_type = local.schema_entity_type
}

resource "confluent_business_metadata_binding" "schema-bm-binding" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  business_metadata_name = confluent_business_metadata.main.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:.:${confluent_schema.purchase.schema_identifier}"
  entity_type = local.schema_entity_type
  attributes = {
    "Team_owner" = "Sam"
    "Slack_contact" = "@sam"
  }
}

# Apply the Tag on a record
resource "confluent_tag_binding" "pii-record-tagging" {
  schema_registry_cluster {
    id = confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = confluent_schema_registry_cluster.main.rest_endpoint
  credentials {
    key    = confluent_api_key.env-manager-schema-registry-api-key.id
    secret = confluent_api_key.env-manager-schema-registry-api-key.secret
  }

  tag_name = confluent_tag.pii.name
  entity_name = "${confluent_schema_registry_cluster.main.id}:.:${confluent_schema.purchase.schema_identifier}:${var.schema_namespace}.${var.record_name}"
  entity_type = local.record_entity_type
}

locals {
  topic_entity_type = "kafka_topic"
  schema_entity_type = "sr_schema"
  record_entity_type = "sr_record"
}
