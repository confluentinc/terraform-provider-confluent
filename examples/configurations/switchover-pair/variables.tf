variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also configurable via the CONFLUENT_CLOUD_API_KEY environment variable)."
  type        = string
  sensitive   = true
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret (also configurable via the CONFLUENT_CLOUD_API_SECRET environment variable)."
  type        = string
  sensitive   = true
}

variable "environment_crn" {
  description = "The CRN of the environment the switchover pair belongs to (e.g. crn://confluent.cloud/organization=.../environment=env-abc123)."
  type        = string
}

variable "west_cluster_crn" {
  description = "The CRN of the Kafka cluster for the 'west' member (e.g. crn://confluent.cloud/organization=.../environment=env-111111/cloud-cluster=lkc-111111)."
  type        = string
}

variable "east_cluster_crn" {
  description = "The CRN of the Kafka cluster for the 'east' member (e.g. crn://confluent.cloud/organization=.../environment=env-222222/cloud-cluster=lkc-222222)."
  type        = string
}
