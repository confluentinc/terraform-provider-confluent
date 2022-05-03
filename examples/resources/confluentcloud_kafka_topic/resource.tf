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

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.basic-cluster.id
  }
  topic_name = "orders"
  partitions_count = 4
  http_endpoint = confluent_kafka_cluster.basic-cluster.http_endpoint
  config = {
    "cleanup.policy" = "compact"
    "max.message.bytes" = "12345"
    "retention.ms" = "67890"
  }
  credentials {
    key = "<Kafka API Key for confluent_kafka_cluster.basic-cluster>"
    secret = "<Kafka API Secret for confluent_kafka_cluster.basic-cluster>"
  }
}
