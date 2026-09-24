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

variable "west_access_point_crn" {
  description = "The CRN of the PrivateLink access point that reaches the 'west' cluster (e.g. crn://confluent.cloud/organization=.../environment=env-111111/gateway=platt-111111/access-point=plattc-111111). Use network_crn in the endpoint filter instead for Dedicated clusters on a Confluent-managed network."
  type        = string
}

variable "east_access_point_crn" {
  description = "The CRN of the PrivateLink access point that reaches the 'east' cluster (e.g. crn://confluent.cloud/organization=.../environment=env-222222/gateway=platt-222222/access-point=plattc-222222)."
  type        = string
}

variable "failover_target" {
  description = "The member to promote to active (e.g. 'east'). Leave unset (null) for normal applies; set it to trigger a failover. Change it again to trigger a subsequent failover or switchback."
  type        = string
  default     = null
}

variable "failover_type" {
  description = "The failover semantics to apply: PLANNED (graceful, waits for replication lag to reach zero), UNPLANNED (immediate), or RESTORE (re-establish the cluster link after an unplanned failover)."
  type        = string
  default     = "PLANNED"

  validation {
    condition     = contains(["PLANNED", "UNPLANNED", "RESTORE"], var.failover_type)
    error_message = "failover_type must be one of PLANNED, UNPLANNED, or RESTORE."
  }
}
