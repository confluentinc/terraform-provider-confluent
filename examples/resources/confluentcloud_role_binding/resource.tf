resource "confluent_service_account" "test_sa" {
  display_name = "test_sa"
  description  = "description for test_sa"
}

resource "confluent_environment" "test-env" {
  display_name = "Development"
}

resource "confluent_role_binding" "env-example-rb" {
  principal   = "User:${confluent_service_account.test_sa.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = confluent_environment.test-env.resource_name
}

resource "confluent_kafka_cluster" "standard-cluster-on-aws" {
  display_name = "standard_kafka_cluster_on_aws"
  availability = "SINGLE_ZONE"
  cloud        = "AWS"
  region       = "us-west-2"
  standard {}

  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_role_binding" "cluster-example-rb" {
  principal   = "User:${confluent_service_account.test_sa.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.standard-cluster-on-aws.rbac_crn
}

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.standard-cluster-on-aws.id
  }
  topic_name       = "orders"
  partitions_count = 4
  http_endpoint    = confluent_kafka_cluster.standard-cluster-on-aws.http_endpoint
  config = {
    "cleanup.policy"    = "compact"
    "max.message.bytes" = "12345"
    "retention.ms"      = "6789000"
  }
  credentials {
    key    = var.kafka_api_key
    secret = var.kafka_api_secret
  }
}

resource "confluent_role_binding" "topic-example-rb" {
  principal   = "User:${confluent_service_account.test_sa.id}"
  role_name   = "DeveloperWrite"
  crn_pattern = "${confluent_kafka_cluster.standard-cluster-on-aws.rbac_crn}/kafka=${confluent_kafka_cluster.standard-cluster-on-aws.id}/topic=${confluent_kafka_topic.orders.topic_name}"
}

resource "confluent_network" "privatelink" {
  display_name     = "Private Link Network"
  cloud            = "AWS"
  region           = "us-east-2"
  connection_types = ["PRIVATELINK"]
  environment {
    id = confluent_environment.test-env.id
  }
}

resource "confluent_role_binding" "network-example-rb" {
  principal   = "User:${confluent_service_account.test_sa.id}"
  role_name   = "NetworkAdmin"
  crn_pattern = confluent_network.privatelink.resource_name
}
