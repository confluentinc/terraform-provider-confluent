module "confluent_kafka_topic_api_keys" {
  source = "./confluent_kafka_topic_api_keys_module"

  # Specifying Cloud API Keys is still necessary for now when managing confluent_kafka_acl
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  kafka_id         = var.kafka_id
  environment_id   = var.environment_id
  topic_name       = var.topic_name
}
