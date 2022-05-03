resource "confluent_environment" "test-env" {
  display_name = "Development"
}

resource "confluent_kafka_cluster" "basic-cluster" {
  display_name = "basic_kafka_cluster"
  availability = "SINGLE_ZONE"
  cloud = "GCP"
  region = "us-central1"
  basic {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_kafka_acl" "describe-basic-cluster" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic-cluster.id
  }
  resource_type = "CLUSTER"
  resource_name = "kafka-cluster"
  pattern_type = "LITERAL"
  principal = "User:sa-xyz123"
  host = "*"
  operation = "DESCRIBE"
  permission = "ALLOW"
  http_endpoint = confluent_kafka_cluster.basic-cluster.http_endpoint
  credentials {
    key = "<Kafka API Key for confluent_kafka_cluster.basic-cluster>"
    secret = "<Kafka API Secret for confluent_kafka_cluster.basic-cluster>"
  }
}
