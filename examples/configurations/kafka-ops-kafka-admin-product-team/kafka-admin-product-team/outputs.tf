output "resource-ids" {
  value = <<-EOT
  Kafka Cluster ID: ${data.confluent_kafka_cluster_v2.basic.id}
  Kafka topic name: ${confluent_kafka_topic_v3.orders.topic_name}

  Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
  ${data.confluent_service_account_v2.app-producer.display_name}:                    ${data.confluent_service_account_v2.app-producer.id}
  ${data.confluent_service_account_v2.app-producer.display_name}'s Kafka API Key:    "${confluent_api_key_v2.app-producer-kafka-api-key.id}"
  ${data.confluent_service_account_v2.app-producer.display_name}'s Kafka API Secret: "${confluent_api_key_v2.app-producer-kafka-api-key.secret}"

  ${data.confluent_service_account_v2.app-consumer.display_name}:                    ${data.confluent_service_account_v2.app-consumer.id}
  ${data.confluent_service_account_v2.app-consumer.display_name}'s Kafka API Key:    "${confluent_api_key_v2.app-consumer-kafka-api-key.id}"
  ${data.confluent_service_account_v2.app-consumer.display_name}'s Kafka API Secret: "${confluent_api_key_v2.app-consumer-kafka-api-key.secret}"

  In order to use the Confluent CLI v2 to produce and consume messages from topic '${confluent_kafka_topic_v3.orders.topic_name}' using Kafka API Keys
  of ${data.confluent_service_account_v2.app-producer.display_name} and ${data.confluent_service_account_v2.app-consumer.display_name} service accounts
  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Produce key-value records to topic '${confluent_kafka_topic_v3.orders.topic_name}' by using ${data.confluent_service_account_v2.app-producer.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${confluent_kafka_topic_v3.orders.topic_name} --environment ${data.confluent_environment_v2.staging.id} --cluster ${data.confluent_kafka_cluster_v2.basic.id} --api-key "${confluent_api_key_v2.app-producer-kafka-api-key.id}" --api-secret "${confluent_api_key_v2.app-producer-kafka-api-key.secret}"
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"number":1,"date":18500,"shipping_address":"899 W Evelyn Ave, Mountain View, CA 94041, USA","cost":15.00}
  # {"number":2,"date":18501,"shipping_address":"1 Bedford St, London WC2E 9HG, United Kingdom","cost":5.00}
  # {"number":3,"date":18502,"shipping_address":"3307 Northland Dr Suite 400, Austin, TX 78731, USA","cost":10.00}

  # 3. Consume records from topic '${confluent_kafka_topic_v3.orders.topic_name}' by using ${data.confluent_service_account_v2.app-consumer.display_name}'s Kafka API Key
  $ confluent kafka topic consume ${confluent_kafka_topic_v3.orders.topic_name} --from-beginning --environment ${data.confluent_environment_v2.staging.id} --cluster ${data.confluent_kafka_cluster_v2.basic.id} --api-key "${confluent_api_key_v2.app-consumer-kafka-api-key.id}" --api-secret "${confluent_api_key_v2.app-consumer-kafka-api-key.secret}"
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
