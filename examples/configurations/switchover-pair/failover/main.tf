terraform {
  required_providers {
    confluent = {
      source = "confluentinc/confluent"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

# This workspace holds the DR *action*: triggering a failover (or switchback) on the
# pair managed by the sibling `infra` workspace. It is kept separate on purpose. A
# failover is an emergency operation an on-call engineer runs under pressure, possibly
# with different credentials or while the infrastructure pipeline is broken; separate
# state keeps the blast radius of each apply matched to its intent — a routine
# infrastructure apply cannot re-fire a failover, and an emergency failover does not
# require the whole infrastructure plan to be clean first.
#
# The pair id and environment CRN can come from either place:
#
#   * passed in directly (`-var switchover_pair_id=sw-abc123 -var environment_crn=crn://…`),
#     which needs nothing but the pair id — useful when the infra state lives elsewhere
#     or is unreachable during an incident; or
#   * read from the infra workspace's state (the default when the variables are unset).
#     With a remote backend, point this data source at that backend instead.
data "terraform_remote_state" "infra" {
  count = var.switchover_pair_id == null || var.environment_crn == null ? 1 : 0

  backend = "local"
  config = {
    path = "${path.module}/../infra/terraform.tfstate"
  }
}

locals {
  switchover_pair_id = coalesce(var.switchover_pair_id, try(data.terraform_remote_state.infra[0].outputs.switchover_pair_id, null))
  environment_crn    = coalesce(var.environment_crn, try(data.terraform_remote_state.infra[0].outputs.environment_crn, null))
}

# Applying this workspace performs the failover:
#
#   terraform apply -var active_member=east                                # PLANNED failover to east
#   terraform apply -var active_member=west                                # later: fail back
#   terraform apply -var active_member=east -var failover_type=UNPLANNED   # immediate failover
#   terraform apply -var failover_type=RESTORE                             # after an UNPLANNED failover; no active_member
#   terraform apply -var switchover_pair_id=sw-abc123 -var environment_crn=crn://… -var active_member=east
#                                                                          # same, naming the pair directly
#
# Every argument is ForceNew, so a changed value recreates the resource, which
# re-triggers the operation. Destroying it only removes it from state; a failover
# cannot be undone by deleting it.
resource "confluent_switchover_pair_failover" "example" {
  switchover_pair_id = local.switchover_pair_id
  active_member      = var.active_member
  failover_type      = var.failover_type
  environment_crn    = local.environment_crn
}

output "failover_phase" {
  description = "The pair's phase right after the failover was triggered (UPDATING); poll the pair for READY_TO_FAILOVER."
  value       = confluent_switchover_pair_failover.example.phase
}
