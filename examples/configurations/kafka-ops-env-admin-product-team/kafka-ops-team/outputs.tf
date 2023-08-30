output "resource-ids" {
  value = <<-EOT
  Environment ID:   ${confluent_environment.staging.id}

  Service Account with EnvironmentAdmin and AccountAdmin roles and its Cloud API Key (API Keys inherit the permissions granted to the owner):

  ${confluent_service_account.env-manager.display_name}:                     ${confluent_service_account.env-manager.id}
  ${confluent_service_account.env-manager.display_name}'s Cloud API Key:     "${confluent_api_key.env-manager-cloud-api-key.id}"
  ${confluent_service_account.env-manager.display_name}'s Cloud API Secret:  "${confluent_api_key.env-manager-cloud-api-key.secret}"

  EOT

  sensitive = true
}
