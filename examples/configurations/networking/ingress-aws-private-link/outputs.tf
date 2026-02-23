output "gateway" {
  description = "The AWS Ingress Private Link Gateway"
  value       = confluent_gateway.main
}

output "access_point" {
  description = "The AWS Ingress Private Link Access Point"
  value       = confluent_access_point.main
}
