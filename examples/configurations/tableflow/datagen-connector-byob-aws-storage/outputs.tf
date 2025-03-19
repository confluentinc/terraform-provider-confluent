output "resource-ids" {
  value = <<-EOT
  You have completed steps #1-3 from https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html

  Environment ID:   ${confluent_environment.staging.id}
  Kafka Cluster ID: ${confluent_kafka_cluster.standard.id}
  Kafka topic name: ${confluent_kafka_topic.stock-trades.topic_name}

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

  Step 4: Set up access to the Iceberg REST Catalog:

  Kafka Cluster ID: ${confluent_kafka_cluster.standard.id}
  REST Catalog Endpoint: "https://tableflow.${var.aws_region}.aws.confluent.cloud/iceberg/catalog/organizations/${data.confluent_organization.main.id}/environments/${confluent_environment.staging.id}"

  Service Account with "EnvironmentAdmin" role its Tableflow API Key (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account.app-reader.display_name}:                        ${confluent_service_account.app-reader.id}
  ${confluent_service_account.app-reader.display_name}'s Tableflow API Key:    "${confluent_api_key.app-reader-tableflow-api-key.id}"
  ${confluent_service_account.app-reader.display_name}'s Tableflow API Secret: "${confluent_api_key.app-reader-tableflow-api-key.secret}"

  Step 5: Query Iceberg tables
  Follow https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-custom-storage-glue.html#step-5-query-iceberg-tables for the remaining steps.

  EOT

  sensitive = true
}
