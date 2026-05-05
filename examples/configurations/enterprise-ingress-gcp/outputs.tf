output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${confluent_environment.main.id}
  Kafka Cluster ID: ${confluent_kafka_cluster.enterprise.id}

  Gateway:
    Gateway ID:            ${confluent_gateway.gcp_ingress.id}
    Service Attachment:    ${confluent_gateway.gcp_ingress.gcp_ingress_private_service_connect_gateway[0].private_service_connect_service_attachment}

  Access Point:
    Access Point ID:                          ${confluent_access_point.gcp_ingress.id}
    Private Service Connect Connection ID:    ${confluent_access_point.gcp_ingress.gcp_ingress_private_service_connect_endpoint[0].private_service_connect_connection_id}
    Private Service Connect Service Attachment: ${confluent_access_point.gcp_ingress.gcp_ingress_private_service_connect_endpoint[0].private_service_connect_service_attachment}
    DNS Domain:                               ${confluent_access_point.gcp_ingress.gcp_ingress_private_service_connect_endpoint[0].dns_domain}

  Service Account:
    ${confluent_service_account.app-manager.display_name}: ${confluent_service_account.app-manager.id}
    Kafka API Key:    "${confluent_api_key.app-manager-kafka-api-key.id}"
    Kafka API Secret: "${confluent_api_key.app-manager-kafka-api-key.secret}"

  To create the GCP Private Service Connect endpoint that connects to this gateway, use:
    Service Attachment: ${confluent_gateway.gcp_ingress.gcp_ingress_private_service_connect_gateway[0].private_service_connect_service_attachment}

  EOT

  sensitive = true
}
