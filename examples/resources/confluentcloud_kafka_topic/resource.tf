resource "confluentcloud_environment" "test-env" {
  display_name = "Development"
}

resource "confluentcloud_kafka_cluster" "basic-cluster" {
  display_name = "basic_kafka_cluster"
  availability = "SINGLE_ZONE"
  cloud = "GCP"
  region = "us-central1"
  basic {}

  environment {
    id = confluentcloud_environment.test-env.id
  }
}

resource "confluentcloud_kafka_topic" "orders" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.basic-cluster.id
  }
  topic_name = "orders"
  partitions_count = 4
  http_endpoint = confluentcloud_kafka_cluster.basic-cluster.http_endpoint
  config = {
    "cleanup.policy" = "compact"
    "max.message.bytes" = "12345"
    "retention.ms" = "67890"
  }
  credentials {
    key = "<Kafka API Key for confluentcloud_kafka_cluster.basic-cluster>"
    secret = "<Kafka API Secret for confluentcloud_kafka_cluster.basic-cluster>"
  }
}
