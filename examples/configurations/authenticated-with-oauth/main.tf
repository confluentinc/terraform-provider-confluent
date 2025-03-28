terraform {
  required_providers {
    confluent = {
      source = "terraform.confluent.io/confluentinc/confluent"
      # source  = "confluentinc/confluent"
      # version = "2.16.0"
    }
  }
}

provider "confluent" {
  cloud_api_key = "VBMLJNNQSG7LHEVN"
  cloud_api_secret = "bnVeT4W1wBtxgaNIZ6WjdtdT1yoRS08NjBXv+LIc5yH7aS6qM+jZetBXQMJTjBJ+"
  oauth {
    oauth_external_token_url = "https://ccloud-sso-sandbox.okta.com/oauth2/ausod37qoaxy2xfjI697/v1/token"
    oauth_external_client_id  = "0oaod69tu7yYnbrMn697"
    oauth_external_client_secret = "QN83b_1JscAAep7JTEdXvdloEUqKBXwk4_K00VyzXnYYBVFbCxdZN4Vy6NUDdo04"
    oauth_identity_pool_id = "pool-W5Qe"
  }
}

resource "confluent_environment" "oauth-demo" {
  display_name = "OAuth_Demo_Environment"

  stream_governance {
    package = "ESSENTIALS"
  }
}

# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster
resource "confluent_kafka_cluster" "basic" {
  display_name = "basic_cluster_oauth_demo"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-east-2"
  basic {}
  environment {
    id = confluent_environment.oauth-demo.id
  }
}

data "confluent_schema_registry_cluster" "main" {
  environment {
    id = confluent_environment.oauth-demo.id
  }

  depends_on = [
    confluent_kafka_cluster.basic
  ]
}

resource "confluent_kafka_topic" "purchase" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  topic_name    = "purchase"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
}

resource "confluent_kafka_topic" "order" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic.id
  }
  topic_name    = "order"
  rest_endpoint = confluent_kafka_cluster.basic.rest_endpoint
}

resource "confluent_schema" "order" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.main.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.main.rest_endpoint
  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.order.topic_name}-value"
  format       = "AVRO"
  schema       = file("./schemas/avro/order.avsc")
}

data "confluent_flink_region" "main" {
  cloud  = "AWS"
  region = "us-east-2"
}

resource "confluent_flink_statement" "read-orders-source-table" {
  organization {
    id = "1771d910-6621-41e2-92ef-1529e7e97410"
  }
  environment {
    id = confluent_environment.oauth-demo.id
  }
  compute_pool {
    id = "lfcp-yx7zqo"
  }
  principal {
    id = "pool-W5Qe"
  }
  # https://docs.confluent.io/cloud/current/flink/reference/example-data.html#marketplace-database
  statement = file("./statements/query-orders-source-table.sql")
  properties = {
    "sql.current-catalog"  = confluent_environment.oauth-demo.display_name
    "sql.current-database" = confluent_kafka_cluster.basic.display_name
  }
  rest_endpoint = data.confluent_flink_region.main.rest_endpoint

  depends_on = [
    confluent_schema.order
  ]
}
