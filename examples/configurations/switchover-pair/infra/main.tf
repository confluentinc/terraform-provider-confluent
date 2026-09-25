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

# This workspace holds the DR *infrastructure*: the switchover pair and its endpoint.
# It is applied on a schedule by the platform team. Failovers live in the sibling
# `failover` workspace so that a routine apply here can never trigger one, and an
# emergency failover does not depend on this plan being clean.

# A switchover pair models a cluster-level DR pairing between two Kafka clusters
# (an active member and a passive member) for disaster recovery. References are
# supplied as full CRNs: each member's CRN carries its own environment, so the two
# members may live in different environments than the pair itself.
#
# `active_member` chooses the side that is active when the pair is created. The
# Switchover service owns it from then on: the resource reads the current value
# back after a failover without planning any change, and editing it here has no
# effect. Use the `failover` workspace to move traffic.
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

# Consumed by the `failover` workspace (via terraform_remote_state or copied into its
# variables), so the pair id never has to be typed by hand.
output "switchover_pair_id" {
  value = confluent_switchover_pair.example.id
}

output "environment_crn" {
  value = var.environment_crn
}

output "switchover_pair_active_member" {
  value = confluent_switchover_pair.example.active_member
}

output "switchover_pair_phase" {
  value = confluent_switchover_pair.example.phase
}

output "switchover_endpoint_target" {
  value = confluent_switchover_endpoint.example.target
}

output "switchover_endpoint_hostnames" {
  value = { for endpoint in confluent_switchover_endpoint.example.endpoints : endpoint.name => endpoint.hostname }
}

# Everything the service resolved for each side, keyed by endpoint name.
output "switchover_endpoints" {
  value = {
    for endpoint in confluent_switchover_endpoint.example.endpoints : endpoint.name => {
      hostname         = endpoint.hostname
      cloud            = endpoint.cloud
      region           = endpoint.region
      connection_type  = endpoint.connection_type
      type             = endpoint.endpoint_filter[0].type
      access_point_crn = endpoint.endpoint_filter[0].access_point_crn
      network_crn      = endpoint.endpoint_filter[0].network_crn
    }
  }
}
