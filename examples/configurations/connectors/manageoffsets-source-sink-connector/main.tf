terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.12.0"
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
    confluent_kafka_cluster.basic
  ]
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

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

// 'app-manager2' service account is required in this configuration to create 'orders' topic and grant ACLs
// to 'app-producer2' and 'app-consumer2' service accounts.
resource "confluent_service_account" "app-manager2" {
  display_name = "app-manager2"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "app-manager2-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager2.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.basic.rbac_crn
}

resource "confluent_api_key" "app-manager2-kafka-api-key" {
  display_name = "app-manager2-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager2' service account"
  owner {
    id          = confluent_service_account.app-manager2.id
    api_version = confluent_service_account.app-manager2.api_version
    kind        = confluent_service_account.app-manager2.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.basic.id
    api_version = confluent_kafka_cluster.basic.api_version
    kind        = confluent_kafka_cluster.basic.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager2-kafka-cluster-admin is created before
  # confluent_api_key.app-manager2-kafka-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager2-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager2-kafka-cluster-admin
  ]
}

resource "confluent_service_account" "app-consumer2" {
  display_name = "app-consumer2"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_service_account" "app-connector" {
  display_name = "app-connector"
  description  = "Service account of mongo db Source Connector to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-consumer2-kafka-api-key" {
  display_name = "app-consumer2-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer2' service account"
  owner {
    id          = confluent_service_account.app-consumer2.id
    api_version = confluent_service_account.app-consumer2.api_version
    kind        = confluent_service_account.app-consumer2.kind
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

resource "confluent_service_account" "app-producer2" {
  display_name = "app-producer2"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-producer2-kafka-api-key" {
  display_name = "app-producer2-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer2' service account"
  owner {
    id          = confluent_service_account.app-producer2.id
    api_version = confluent_service_account.app-producer2.api_version
    kind        = confluent_service_account.app-producer2.kind
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


resource "confluent_kafka_acl" "app-connector-describe-on-cluster" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "CLUSTER"
  resource_name = "kafka-cluster"
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "DESCRIBE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-create-on-prefix-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = var.mongodb_topic_prefix
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "CREATE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-write-on-prefix-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = var.mongodb_topic_prefix
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-create-on-data-preview-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "data-preview.${var.mongodb_database}.${var.mongodb_collection}"
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "CREATE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-write-on-data-preview-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "data-preview.${var.mongodb_database}.${var.mongodb_collection}"
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-read-on-target-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-create-on-dlq-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "dlq-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "CREATE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-write-on-dlq-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "dlq-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-create-on-success-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "success-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "CREATE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-write-on-success-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "success-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-create-on-error-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "error-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "CREATE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-write-on-error-lcc-topics" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "TOPIC"
  resource_name = "error-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-connector-read-on-connect-lcc-group" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  resource_type = "GROUP"
  resource_name = "connect-lcc"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-connector.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager2-kafka-api-key.id
    secret = confluent_api_key.app-manager2-kafka-api-key.secret
  }
}

resource "confluent_connector" "mysql-db-sink" {
  environment {
    id = confluent_environment.staging.id
  }

  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }

  config_sensitive = {
    "connection.password" = var.mysqldb_password
  }

  config_nonsensitive = {
    "connector.class"          = "MySqlSink"
    "name"                     = "MySQLSinkConnector_0"
    "topics"                   = var.mysqldb_topic_name
    "input.data.format"        = "AVRO"
    "tasks.max"                = "1"
    "db.name"                  = var.mysqldb_name
    "insert.mode"              = "INSERT"
    "auto.create"              = "true"
    "auto.evolve"              = "true"
    "kafka.auth.mode"          = "SERVICE_ACCOUNT"
    "kafka.service.account.id" = confluent_service_account.app-connector.id
    "connection.user"          = var.mysqldb_user
    "connection.host"          = var.mysqldb_host
    "connection.port"          = var.mysqldb_port
    "ssl.mode"                 = "prefer"
  }

  offsets {
    partition = {
      "kafka_partition" = 0,
      "kafka_topic"     = var.mysqldb_topic_name
    }
    offset = {
      "kafka_offset" = 10
    }
  }
  offsets {
    partition = {
      "kafka_partition" = 1,
      "kafka_topic"     = var.mysqldb_topic_name
    }
    offset = {
      "kafka_offset" = 10
    }
  }
  offsets {
    partition = {
      "kafka_partition" = 2,
      "kafka_topic"     = var.mysqldb_topic_name
    }
    offset = {
      "kafka_offset" = 15
    }
  }

  depends_on = [
    confluent_kafka_acl.app-connector-describe-on-cluster,
    confluent_kafka_acl.app-connector-read-on-target-topic,
    confluent_kafka_acl.app-connector-create-on-dlq-lcc-topics,
    confluent_kafka_acl.app-connector-write-on-dlq-lcc-topics,
    confluent_kafka_acl.app-connector-create-on-success-lcc-topics,
    confluent_kafka_acl.app-connector-write-on-success-lcc-topics,
    confluent_kafka_acl.app-connector-create-on-error-lcc-topics,
    confluent_kafka_acl.app-connector-write-on-error-lcc-topics,
    confluent_kafka_acl.app-connector-read-on-connect-lcc-group,
  ]
}

resource "confluent_connector" "mongo-db-source" {
  environment {
    id = confluent_environment.staging.id
  }

  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }

  // Block for custom *sensitive* configuration properties that are labelled with "Type: password" under "Configuration Properties" section in the docs:
  // https://docs.confluent.io/cloud/current/connectors/cc-mongo-db-source.html#configuration-properties
  config_sensitive = {
    "connection.password" = var.mongodb_password
  }

  // Block for custom *nonsensitive* configuration properties that are *not* labelled with "Type: password" under "Configuration Properties" section in the docs:
  // https://docs.confluent.io/cloud/current/connectors/cc-mongo-db-source.html#configuration-properties
  config_nonsensitive = {
    "connector.class"          = "MongoDbAtlasSource"
    "name"                     = "MongoDbAtlasSourceConnector"
    "kafka.auth.mode"          = "SERVICE_ACCOUNT"
    "kafka.service.account.id" = confluent_service_account.app-connector.id
    "topic.prefix"             = var.mongodb_topic_prefix
    "database"                 = var.mongodb_database
    "collection"               = var.mongodb_collection
    "poll.await.time.ms"       = "5000"
    "poll.max.batch.size"      = "1000"
    "copy.existing"            = "true"
    "output.data.format"       = "JSON"
    "tasks.max"                = "1"
    "connection.host"          = var.mongodb_connection_host
    "connection.user"          = var.mongodb_connection_user
  }

  offsets {
    partition = {
      "ns" = "mongodb+srv://testcluster.wy6ey.mongodb.net/sample_mflix.movies"
    }
    offset = {
      "_id"  = "{\"_id\": {\"$oid\": \"573a1392f29313caabcd9cce\"}, \"copyingData\": true}"
      "copy" = "true"
    }
  }

  depends_on = [
    confluent_kafka_acl.app-connector-describe-on-cluster,
    confluent_kafka_acl.app-connector-create-on-prefix-topics,
    confluent_kafka_acl.app-connector-write-on-prefix-topics,
    confluent_kafka_acl.app-connector-create-on-data-preview-topics,
    confluent_kafka_acl.app-connector-write-on-data-preview-topics,
  ]
}
