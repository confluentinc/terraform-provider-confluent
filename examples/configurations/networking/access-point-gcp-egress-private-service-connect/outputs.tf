output "access_point" {
  description = "The GCP Egress Private Service Connect Access Point"
  value       = confluent_access_point.private-service-connect
}
