output "gateway" {
  description = "The AWS Private Network Interface Gateway"
  value       = confluent_gateway.main
}

output "access_point" {
  description = "The AWS Private Network Interface Access Point"
  value       = confluent_access_point.main
}
