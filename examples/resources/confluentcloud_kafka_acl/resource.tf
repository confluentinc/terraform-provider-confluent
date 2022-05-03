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

resource "confluentcloud_kafka_acl" "describe-basic-cluster" {
  kafka_cluster {
    id = confluentcloud_kafka_cluster.basic-cluster.id
  }
  resource_type = "CLUSTER"
  resource_name = "kafka-cluster"
  pattern_type = "LITERAL"
  principal = "User:sa-xyz123"
  host = "*"
  operation = "DESCRIBE"
  permission = "ALLOW"
  http_endpoint = confluentcloud_kafka_cluster.basic-cluster.http_endpoint
  credentials {
    key = "<Kafka API Key for confluentcloud_kafka_cluster.basic-cluster>"
    secret = "<Kafka API Secret for confluentcloud_kafka_cluster.basic-cluster>"
  }
}
