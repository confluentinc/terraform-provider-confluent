variable "schema_registry_api_key" {
  description = "Schema Registry API Key"
  type        = string
  sensitive   = true
}

variable "schema_registry_api_secret" {
  description = "Schema Registry API Secret"
  type        = string
  sensitive   = true
}

variable "schema_registry_rest_endpoint" {
  description = "The REST Endpoint of the Schema Registry cluster"
  type        = string
}

variable "schema_registry_id" {
  description = "The ID the the Schema Registry cluster of the form 'lsrc-'"
  type        = string
}
