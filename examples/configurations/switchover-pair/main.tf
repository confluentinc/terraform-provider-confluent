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

# A switchover pair models a cluster-level DR pairing between two Kafka clusters
# (an active member and a passive member) for disaster recovery. References are
# supplied as full CRNs: each member's CRN carries its own environment, so the two
# members may live in different environments than the pair itself.
#
# `active_member` chooses the side that is active when the pair is created. The
# Switchover service owns it from then on: the resource reads the current value
# back after a failover without planning any change, and editing it here has no
# effect. Use the failover resource below to move traffic.
resource "confluent_switchover_pair" "example" {
  display_name  = "prod-kafka-dr"
  active_member = "west"

  members {
    name       = "west"
    member_crn = var.west_cluster_crn
  }

  members {
    name       = "east"
    member_crn = var.east_cluster_crn
  }

  environment_crn = var.environment_crn
}

# A switchover endpoint is a sticky DR bootstrap bound to the pair. Its computed
# `target` starts on the side matching the pair's active member and follows it
# across failovers.
resource "confluent_switchover_endpoint" "example" {
  display_name        = "prod-kafka-dr-endpoint"
  parent_resource_crn = "${var.environment_crn}/switchover-pair=${confluent_switchover_pair.example.id}"

  endpoints {
    name = "west-platt"
    endpoint_filter {
      type             = "private"
      access_point_crn = var.west_access_point_crn
    }
  }

  endpoints {
    name = "east-platt"
    endpoint_filter {
      type             = "private"
      access_point_crn = var.east_access_point_crn
    }
  }
}

# Failover is an imperative operation, so it is modeled as an action resource that
# lives in the same configuration as the pair and endpoint. Leave `failover_target`
# unset for day-to-day applies; set it (for example `-var failover_target=east`) to
# trigger a failover. Because the pair and endpoint no longer treat the active member
# or target as inputs, a failover does not cause either of them to be replaced.
#
# To trigger another failover later, change `failover_target` (or `failover_type`);
# all inputs on the failover resource are ForceNew, so the change re-runs the operation.
resource "confluent_switchover_pair_failover" "example" {
  count = var.failover_target == null ? 0 : 1

  switchover_pair_id = confluent_switchover_pair.example.id
  active_member      = var.failover_target
  failover_type      = var.failover_type
  environment_crn    = var.environment_crn

  # The endpoint must exist before a failover can be triggered.
  depends_on = [confluent_switchover_endpoint.example]
}

output "switchover_pair_phase" {
  value = confluent_switchover_pair.example.phase
}

output "switchover_pair_active_member" {
  value = confluent_switchover_pair.example.active_member
}

output "switchover_endpoint_target" {
  value = confluent_switchover_endpoint.example.target
}

output "switchover_endpoint_hostnames" {
  value = { for endpoint in confluent_switchover_endpoint.example.endpoints : endpoint.name => endpoint.hostname }
}
