output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${confluent_environment.main.id}
  Kafka Cluster ID: ${confluent_kafka_cluster.enterprise.id}

  Gateway:
    Gateway ID:                       ${confluent_gateway.azure_ingress.id}
    Private Link Service Alias:       ${confluent_gateway.azure_ingress.azure_ingress_private_link_gateway[0].private_link_service_alias}
    Private Link Service Resource ID: ${confluent_gateway.azure_ingress.azure_ingress_private_link_gateway[0].private_link_service_resource_id}

  Access Point:
    Access Point ID:                  ${confluent_access_point.azure_ingress.id}
    Private Endpoint Resource ID:     ${confluent_access_point.azure_ingress.azure_ingress_private_link_endpoint[0].private_endpoint_resource_id}
    Private Link Service Alias:       ${confluent_access_point.azure_ingress.azure_ingress_private_link_endpoint[0].private_link_service_alias}
    Private Link Service Resource ID: ${confluent_access_point.azure_ingress.azure_ingress_private_link_endpoint[0].private_link_service_resource_id}
    DNS Domain:                       ${confluent_access_point.azure_ingress.azure_ingress_private_link_endpoint[0].dns_domain}

  Service Account:
    ${confluent_service_account.app-manager.display_name}: ${confluent_service_account.app-manager.id}
    Kafka API Key:    "${confluent_api_key.app-manager-kafka-api-key.id}"
    Kafka API Secret: "${confluent_api_key.app-manager-kafka-api-key.secret}"

  To create the Azure Private Endpoint that connects to this gateway, use:
    Private Link Service Alias: ${confluent_gateway.azure_ingress.azure_ingress_private_link_gateway[0].private_link_service_alias}

  EOT

  sensitive = true
}
