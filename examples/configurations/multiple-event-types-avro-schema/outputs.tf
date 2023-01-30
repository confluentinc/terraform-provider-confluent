output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${confluent_environment.staging.id}
  Kafka Cluster ID: ${confluent_kafka_cluster.basic.id}
  Kafka topic name: ${confluent_kafka_topic.customer-event.topic_name}

  Service Accounts and their Kafka API Keys (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account.app-manager.display_name}:                     ${confluent_service_account.app-manager.id}
  ${confluent_service_account.app-manager.display_name}'s Kafka API Key:     "${confluent_api_key.app-manager-kafka-api-key.id}"
  ${confluent_service_account.app-manager.display_name}'s Kafka API Secret:  "${confluent_api_key.app-manager-kafka-api-key.secret}"

  ${confluent_service_account.app-producer.display_name}:                    ${confluent_service_account.app-producer.id}
  ${confluent_service_account.app-producer.display_name}'s Kafka API Key:    "${confluent_api_key.app-producer-kafka-api-key.id}"
  ${confluent_service_account.app-producer.display_name}'s Kafka API Secret: "${confluent_api_key.app-producer-kafka-api-key.secret}"

  ${confluent_service_account.app-consumer.display_name}:                    ${confluent_service_account.app-consumer.id}
  ${confluent_service_account.app-consumer.display_name}'s Kafka API Key:    "${confluent_api_key.app-consumer-kafka-api-key.id}"
  ${confluent_service_account.app-consumer.display_name}'s Kafka API Secret: "${confluent_api_key.app-consumer-kafka-api-key.secret}"

  In order to use the Confluent CLI v2 to produce and consume messages from topic '${confluent_kafka_topic.customer-event.topic_name}' using Kafka API Keys
  of ${confluent_service_account.app-producer.display_name} and ${confluent_service_account.app-consumer.display_name} service accounts
  run the following commands:

  # 1. Log in to Confluent Cloud
  $ confluent login

  # 2. Produce key-value records to topic '${confluent_kafka_topic.customer-event.topic_name}' by using ${confluent_service_account.app-producer.display_name}'s Kafka API Key
  $ confluent kafka topic produce ${confluent_kafka_topic.customer-event.topic_name} \
        --schema-id ${confluent_schema.customer-event.schema_identifier} \
        --value-format avro \
        --sr-endpoint ${confluent_schema_registry_cluster.essentials.rest_endpoint} \
        --sr-api-key "${confluent_api_key.env-manager-schema-registry-api-key.id}" \
        --sr-api-secret "${confluent_api_key.env-manager-schema-registry-api-key.secret}" \
        --cluster ${confluent_kafka_cluster.basic.id} \
        --api-key "${confluent_api_key.app-producer-kafka-api-key.id}" \
        --api-secret "${confluent_api_key.app-producer-kafka-api-key.secret}" \
        --environment ${confluent_environment.staging.id}
  # Enter a few records and then press 'Ctrl-C' when you're done.
  # Sample records:
  # {"id":"1", "pageView":{"url":"https://www.confluent.io/","is_special":false,"customer_id":"lombardi"}}
  # {"id":"2", "pageView":{"url":"https://www.confluent.io/","is_special":false,"customer_id":"lombardi"}}
  # {"id":"3", "purchase":{"item":"pizza","amount":0.99,"customer_id":"lombardi"}}

  # 3. Consume records from topic '${confluent_kafka_topic.customer-event.topic_name}' by using ${confluent_service_account.app-consumer.display_name}'s Kafka API Key
  $ confluent kafka topic consume customer-event \
        --from-beginning \
        --value-format avro \
        --sr-endpoint ${confluent_schema_registry_cluster.essentials.rest_endpoint} \
        --sr-api-key "${confluent_api_key.env-manager-schema-registry-api-key.id}" \
        --sr-api-secret "${confluent_api_key.env-manager-schema-registry-api-key.secret}" \
        --cluster ${confluent_kafka_cluster.basic.id} \
        --api-key "${confluent_api_key.app-consumer-kafka-api-key.id}" \
        --api-secret "${confluent_api_key.app-consumer-kafka-api-key.secret}" \
        --environment ${confluent_environment.staging.id}
  # When you are done, press 'Ctrl-C'.
  EOT

  sensitive = true
}
