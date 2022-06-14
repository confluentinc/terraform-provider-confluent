output "resource-ids" {
  value = <<-EOT
  Environment ID:     ${confluent_environment_v2.staging.id}
  Kafka cluster ID:   ${confluent_kafka_cluster_v2.basic.id}

  Service Accounts with CloudClusterAdmin role and their API Keys (API Keys inherit the permissions granted to the owner):
  ${confluent_service_account_v2.app-manager.display_name}:                     ${confluent_service_account_v2.app-manager.id}
  ${confluent_service_account_v2.app-manager.display_name}'s Cloud API Key:     "${confluent_api_key_v2.app-manager-cloud-api-key.id}"
  ${confluent_service_account_v2.app-manager.display_name}'s Cloud API Secret:  "${confluent_api_key_v2.app-manager-cloud-api-key.secret}"

  ${confluent_service_account_v2.app-manager.display_name}'s Kafka API Key:     "${confluent_api_key_v2.app-manager-kafka-api-key.id}"
  ${confluent_service_account_v2.app-manager.display_name}'s Kafka API Secret:  "${confluent_api_key_v2.app-manager-kafka-api-key.secret}"

  Service Accounts with no roles assigned:
  ${confluent_service_account_v2.app-consumer.display_name}:                    ${confluent_service_account_v2.app-consumer.id}
  ${confluent_service_account_v2.app-producer.display_name}:                    ${confluent_service_account_v2.app-producer.id}

  EOT

  sensitive = true
}
