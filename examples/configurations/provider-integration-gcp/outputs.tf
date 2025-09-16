output "provider_integration_id" {
  description = "The ID of the created provider integration"
  value       = confluent_provider_integration_v2.gcp.id
}

output "provider_integration_status" {
  description = "The status of the provider integration"
  value       = confluent_provider_integration_v2.gcp.status
}

output "confluent_service_account" {
  description = "The Confluent Service Account that will impersonate your service account"
  value       = try(confluent_provider_integration_v2_authorization.gcp.gcp[0].google_service_account, "not_available")
}

output "customer_service_account" {
  description = "Your service account that Confluent will impersonate"
  value       = try(confluent_provider_integration_v2_authorization.gcp.gcp[0].customer_google_service_account, "not_available")
}

output "gcp_setup_command" {
  description = "GCP IAM command to grant Service Account Token Creator role"
  value       = "gcloud projects add-iam-policy-binding YOUR_PROJECT_ID --member=\"serviceAccount:${try(confluent_provider_integration_v2_authorization.gcp.gcp[0].google_service_account, "CONFLUENT_SA")}\" --role=\"roles/iam.serviceAccountTokenCreator\" --condition=\"expression=request.auth.claims.sub=='${try(confluent_provider_integration_v2_authorization.gcp.gcp[0].google_service_account, "CONFLUENT_SA")}'\""
}