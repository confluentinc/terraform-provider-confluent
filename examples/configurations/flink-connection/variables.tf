variable "organization_id" {
  description = "The ID of Confluent Cloud organization (for example, foobar). You could find it on XYZ page."
  type        = string
}

variable "environment_id" {
  description = "The ID of the managed environment on Confluent Cloud."
  type        = string
}

variable "flink_compute_pool_id" {
  description = "The ID of the managed Compute Pool on Confluent Cloud."
  type        = string
}

variable "flink_rest_endpoint" {
  description = "The REST endpoint of the target Flink Region on Confluent Cloud."
  type        = string
}

variable "flink_api_key" {
  description = "Flink API Key (also referred as Flink API ID) that should be owned by a principal with a FlinkAdmin role (provided by Ops team)"
  type        = string
}

variable "flink_api_secret" {
  description = "Flink API Secret (provided by Ops team)"
  type        = string
  sensitive   = true
}

# FlinkAdmin principal needs an Assigner role binding on flink_principal_id principal.
# See https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/flink-quickstart/main.tf#L64
variable "flink_principal_id" {
  description = "Service account to perform a task within Confluent Cloud, such as executing a Flink statement."
  type        = string
}
