output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${confluentcloud_environment.staging.id}
  Kafka Cluster ID: ${confluentcloud_kafka_cluster.dedicated.id}
  Kafka topic name: ${confluentcloud_kafka_topic.orders.topic_name}

  Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
  ${confluentcloud_service_account.app-manager.display_name}:                     ${confluentcloud_service_account.app-manager.id}
  ${confluentcloud_service_account.app-manager.display_name}'s Kafka API Key:     "${confluentcloud_api_key.app-manager-kafka-api-key.id}"
  ${confluentcloud_service_account.app-manager.display_name}'s Kafka API Secret:  "${confluentcloud_api_key.app-manager-kafka-api-key.secret}"

  ${confluentcloud_service_account.app-producer.display_name}:                    ${confluentcloud_service_account.app-producer.id}
  ${confluentcloud_service_account.app-producer.display_name}'s Kafka API Key:    "${confluentcloud_api_key.app-producer-kafka-api-key.id}"
  ${confluentcloud_service_account.app-producer.display_name}'s Kafka API Secret: "${confluentcloud_api_key.app-producer-kafka-api-key.secret}"

  ${confluentcloud_service_account.app-consumer.display_name}:                    ${confluentcloud_service_account.app-consumer.id}
  ${confluentcloud_service_account.app-consumer.display_name}'s Kafka API Key:    "${confluentcloud_api_key.app-consumer-kafka-api-key.id}"
  ${confluentcloud_service_account.app-consumer.display_name}'s Kafka API Secret: "${confluentcloud_api_key.app-consumer-kafka-api-key.secret}"

  In order to use the Confluent CLI v2 to produce and consume messages from topic '${confluentcloud_kafka_topic.orders.topic_name}' using Kafka API Keys
  of ${confluentcloud_service_account.app-producer.display_name} and ${confluentcloud_service_account.app-consumer.display_name} service accounts
  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Produce key-value records to topic '${confluentcloud_kafka_topic.orders.topic_name}' by using ${confluentcloud_service_account.app-producer.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${confluentcloud_kafka_topic.orders.topic_name} --environment ${confluentcloud_environment.staging.id} --cluster ${confluentcloud_kafka_cluster.dedicated.id} --api-key "${confluentcloud_api_key.app-producer-kafka-api-key.id}" --api-secret "${confluentcloud_api_key.app-producer-kafka-api-key.secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 3. Consume records from topic '${confluentcloud_kafka_topic.orders.topic_name}' by using ${confluentcloud_service_account.app-consumer.display_name}'s Kafka API Key
  $ confluent kafka topic consume ${confluentcloud_kafka_topic.orders.topic_name} --from-beginning --environment ${confluentcloud_environment.staging.id} --cluster ${confluentcloud_kafka_cluster.dedicated.id} --api-key "${confluentcloud_api_key.app-consumer-kafka-api-key.id}" --api-secret "${confluentcloud_api_key.app-consumer-kafka-api-key.secret}"
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
