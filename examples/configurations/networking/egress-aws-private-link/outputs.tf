output "gateway" {
  description = "The AWS Egress Private Link Gateway"
  value       = confluent_gateway.main
}

output "access_point" {
  description = "The AWS Egress Private Link Access Point"
  value       = confluent_access_point.main
}
