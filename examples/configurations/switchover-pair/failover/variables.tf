variable "confluent_cloud_api_key" {
  description = "Confluent Cloud API Key (also configurable via the CONFLUENT_CLOUD_API_KEY environment variable). May be a different, more privileged key than the one used by the infra workspace."
  type        = string
  sensitive   = true
}

variable "confluent_cloud_api_secret" {
  description = "Confluent Cloud API Secret (also configurable via the CONFLUENT_CLOUD_API_SECRET environment variable)."
  type        = string
  sensitive   = true
}

variable "switchover_pair_id" {
  description = "The ID of the switchover pair to fail over (e.g. sw-abc123). Leave unset to read it from the infra workspace's state."
  type        = string
  default     = null
}

variable "environment_crn" {
  description = "The CRN of the environment that owns the switchover pair (e.g. crn://confluent.cloud/organization=.../environment=env-abc123). Leave unset to read it from the infra workspace's state."
  type        = string
  default     = null
}

variable "active_member" {
  description = "The member to promote to active (for example 'east'). Required for PLANNED and UNPLANNED failovers; must be left unset for RESTORE."
  type        = string
  default     = null
}

variable "failover_type" {
  description = "The failover semantics to apply: PLANNED (graceful, waits for replication lag to reach zero), UNPLANNED (immediate), or RESTORE (re-establish the cluster link after an unplanned failover). Leave unset to keep the last applied value (PLANNED on first create); a default here would re-trigger a PLANNED failover on the first apply after a RESTORE."
  type        = string
  default     = null

  validation {
    condition     = var.failover_type == null || contains(["PLANNED", "UNPLANNED", "RESTORE"], var.failover_type)
    error_message = "failover_type must be one of PLANNED, UNPLANNED, or RESTORE."
  }
}
