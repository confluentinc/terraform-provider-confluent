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
# (an active member and a passive member) for disaster recovery.
resource "confluent_switchover_pair" "example" {
  display_name  = "prod-kafka-dr"
  active_member = "west"

  members {
    name      = "west"
    member_id = var.west_cluster_id
  }

  members {
    name      = "east"
    member_id = var.east_cluster_id
  }

  environment {
    id = var.environment_id
  }
}

data "confluent_switchover_pair" "example" {
  id = confluent_switchover_pair.example.id

  environment {
    id = var.environment_id
  }
}

output "switchover_pair_phase" {
  value = data.confluent_switchover_pair.example.phase
}
