output "consumer_kafka_api_key_id" {
  description = "Kafka API Key ID to consume the data from the target topic"
  value       = confluent_api_key.app-consumer-kafka-api-key.id
}

output "consumer_kafka_api_key_secret" {
  description = "Kafka API Key Secret to consume the data from the target topic"
  value       = confluent_api_key.app-consumer-kafka-api-key.secret
  sensitive   = true
}

output "producer_kafka_api_key_id" {
  description = "Kafka API Key ID to produce the data to the target topic"
  value       = confluent_api_key.app-producer-kafka-api-key.id
}

output "producer_kafka_api_key_secret" {
  description = "Kafka API Key Secret to produce the data to the target topic"
  value       = confluent_api_key.app-producer-kafka-api-key.secret
  sensitive   = true
}

output "environment_id" {
  description = "The ID of the Environment that the Kafka cluster belongs to of the form 'env-'"
  value       = confluent_environment.main.id
}

output "kafka_id" {
  description = "The ID of the Kafka cluster of the form 'lkc-' that has got the target topic"
  value       = confluent_kafka_cluster.main.id
}
