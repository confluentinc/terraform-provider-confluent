output "resource-ids" {
  value = <<-EOT
  You have completed steps #1-2 from https://docs.confluent.io/cloud/current/topics/tableflow/get-started/quick-start-managed-storage.html#step-1-create-a-topic-and-publish-data

  Environment ID:   ${confluent_environment.staging.id}
  Kafka Cluster ID: ${confluent_kafka_cluster.standard.id}
  Kafka Cluster Name: ${confluent_kafka_cluster.standard.display_name}
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

  Step 3: Set up access to the Iceberg REST Catalog:

  REST Catalog Endpoint: "https://tableflow.${var.aws_region}.aws.confluent.cloud/iceberg/catalog/organizations/${data.confluent_organization.main.id}/environments/${confluent_environment.staging.id}"

  Service Account with "ResourceOwner" role for the scope=Cluster and its Tableflow API Key (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account.app-reader.display_name}:                        ${confluent_service_account.app-reader.id}
  ${confluent_service_account.app-reader.display_name}'s Tableflow API Key:    "${confluent_api_key.app-reader-tableflow-api-key.id}"
  ${confluent_service_account.app-reader.display_name}'s Tableflow API Secret: "${confluent_api_key.app-reader-tableflow-api-key.secret}"

  EOT

  sensitive = true
}
