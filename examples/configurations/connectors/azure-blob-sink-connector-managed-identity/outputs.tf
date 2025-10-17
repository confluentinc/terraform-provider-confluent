output "resource_ids" {
  description = "Resource IDs for all created resources"
  value = <<-EOT
  Environment ID:                   ${confluent_environment.staging.id}
  Kafka Cluster ID:                 ${confluent_kafka_cluster.basic.id}
  Kafka topic name:                 ${confluent_kafka_topic.orders.topic_name}
  
  Provider Integration ID:          ${confluent_provider_integration_setup.azure.id}
  Confluent Multi-Tenant App ID:    ${confluent_provider_integration_authorization.azure.azure[0].confluent_multi_tenant_app_id}
  
  Service Account (app-manager):    ${confluent_service_account.app-manager.id}
  Service Account (app-producer):   ${confluent_service_account.app-producer.id}
  Service Account (app-consumer):   ${confluent_service_account.app-consumer.id}
  Service Account (app-connector):  ${confluent_service_account.app-connector.id}
  
  Azure Blob Sink Connector ID:     ${confluent_connector.azure-blob-sink.id}
  Azure Service Principal ID:       ${module.azure_resources.service_principal_client_id}
  EOT
}

output "confluent_multi_tenant_app_id" {
  description = "Confluent Multi-Tenant App ID for Azure setup"
  value       = confluent_provider_integration_authorization.azure.azure[0].confluent_multi_tenant_app_id
}

output "azure_service_principal_client_id" {
  description = "Azure Service Principal Client ID created for Confluent"
  value       = module.azure_resources.service_principal_client_id
}

output "provider_integration_id" {
  description = "Provider Integration ID"
  value       = confluent_provider_integration_setup.azure.id
}

