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

data "confluent_switchover_pair" "example" {
  id              = confluent_switchover_pair.example.id
  environment_crn = var.environment_crn
}

output "switchover_pair_phase" {
  value = data.confluent_switchover_pair.example.phase
}
