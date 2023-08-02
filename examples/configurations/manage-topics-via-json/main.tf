module "confluent_kafka_topics" {
  source = "./confluent_kafka_topics_module"

  kafka_id            = var.kafka_id
  kafka_rest_endpoint = var.kafka_rest_endpoint
  kafka_api_key       = var.kafka_api_key
  kafka_api_secret    = var.kafka_api_secret
  topics              = jsondecode(file("topics.json"))
}
