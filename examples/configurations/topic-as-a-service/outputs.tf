output "consumer_kafka_api_key_id" {
  description = "Kafka API Key ID to consume the data from the target topic"
  value       = module.topic_as_a_service.consumer_kafka_api_key_id
}

output "consumer_kafka_api_key_secret" {
  description = "Kafka API Key Secret to consume the data from the target topic"
  value       = module.topic_as_a_service.consumer_kafka_api_key_secret
  sensitive   = true
}

output "producer_kafka_api_key_id" {
  description = "Kafka API Key ID to produce the data to the target topic"
  value       = module.topic_as_a_service.producer_kafka_api_key_id
}

output "producer_kafka_api_key_secret" {
  description = "Kafka API Key Secret to produce the data to the target topic"
  value       = module.topic_as_a_service.producer_kafka_api_key_secret
  sensitive   = true
}

output "environment_id" {
  description = "The ID of the Environment that the Kafka cluster belongs to of the form 'env-'"
  value       = module.topic_as_a_service.environment_id
}

output "kafka_id" {
  description = "The ID of the Kafka cluster of the form 'lkc-' that has got the target topic"
  value       = module.topic_as_a_service.kafka_id
}

output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${module.topic_as_a_service.environment_id}
  Kafka Cluster ID: ${module.topic_as_a_service.kafka_id}
  Kafka topic name: ${var.topic_name}

  Kafka API Key to consume the data from the '${var.topic_name}' topic:
  Kafka API Key:     "${module.topic_as_a_service.consumer_kafka_api_key_id}"
  Kafka API Secret:  "${module.topic_as_a_service.consumer_kafka_api_key_secret}"

  Kafka API Key to produce the data to the '${var.topic_name}' topic:
  Kafka API Key:     "${module.topic_as_a_service.producer_kafka_api_key_id}"
  Kafka API Secret:  "${module.topic_as_a_service.producer_kafka_api_key_secret}"

  In order to use the Confluent CLI v2 to produce and consume messages from topic '${var.topic_name}' using the generated Kafka API Keys
  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Produce key-value records to topic '${var.topic_name}' by using 'producer' Kafka API Key
  $ confluent kafka topic produce ${var.topic_name} --environment ${module.topic_as_a_service.environment_id} --cluster ${module.topic_as_a_service.kafka_id} --api-key "${module.topic_as_a_service.producer_kafka_api_key_id}" --api-secret "${module.topic_as_a_service.producer_kafka_api_key_secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 3. Consume records from topic '${var.topic_name}' by using 'consumer' Kafka API Key
  $ confluent kafka topic consume ${var.topic_name} --from-beginning --environment ${module.topic_as_a_service.environment_id} --cluster ${module.topic_as_a_service.kafka_id} --api-key "${module.topic_as_a_service.consumer_kafka_api_key_id}" --api-secret "${module.topic_as_a_service.consumer_kafka_api_key_secret}"
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
