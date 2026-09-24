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
# The pair id and environment are read from the infra workspace's state. With a
# remote backend, point this data source at that backend instead (or pass the two
# values in as variables).
data "terraform_remote_state" "infra" {
  backend = "local"
  config = {
    path = "${path.module}/../infra/terraform.tfstate"
  }
}

# Applying this workspace performs the failover:
#
#   terraform apply -var active_member=east                        # PLANNED failover to east
#   terraform apply -var active_member=west                        # later: fail back
#   terraform apply -var active_member=east -var failover_type=UNPLANNED
#   terraform apply -var failover_type=RESTORE -var active_member=east   # after an UNPLANNED failover
#
# Every argument is ForceNew, so a changed value recreates the resource, which
# re-triggers the operation. Destroying it only removes it from state; a failover
# cannot be undone by deleting it.
resource "confluent_switchover_pair_failover" "example" {
  switchover_pair_id = data.terraform_remote_state.infra.outputs.switchover_pair_id
  active_member      = var.active_member
  failover_type      = var.failover_type
  environment_crn    = data.terraform_remote_state.infra.outputs.environment_crn
}

output "failover_phase" {
  description = "The pair's phase right after the failover was triggered (UPDATING); poll the pair for READY_TO_FAILOVER."
  value       = confluent_switchover_pair_failover.example.phase
}
