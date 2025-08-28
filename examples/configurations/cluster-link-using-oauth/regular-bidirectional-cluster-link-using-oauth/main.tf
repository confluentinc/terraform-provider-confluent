terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.38.0"
    }
  }
}

provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url
    oauth_external_client_id  = var.oauth_external_client_id
    oauth_external_client_secret = var.oauth_external_client_secret
    oauth_identity_pool_id = var.oauth_identity_pool_id
  }
}

data "confluent_environment" "east" {
  id = var.east_kafka_cluster_environment_id
}

data "confluent_environment" "west" {
  id = var.west_kafka_cluster_environment_id
}

data "confluent_kafka_cluster" "east" {
  id = var.east_kafka_cluster_id
  environment {
    id = data.confluent_environment.east.id
  }
}

data "confluent_kafka_cluster" "west" {
  id = var.west_kafka_cluster_id
  environment {
    id = data.confluent_environment.west.id
  }
}

resource "confluent_cluster_link" "east-to-west" {
  link_name = var.cluster_link_name
  link_mode = "BIDIRECTIONAL"
  local_kafka_cluster {
    id            = data.confluent_kafka_cluster.east.id
    rest_endpoint = data.confluent_kafka_cluster.east.rest_endpoint
  }

  remote_kafka_cluster {
    id                 = data.confluent_kafka_cluster.west.id
    bootstrap_endpoint = data.confluent_kafka_cluster.west.bootstrap_endpoint
  }
}

resource "confluent_cluster_link" "west-to-east" {
  link_name = var.cluster_link_name
  link_mode = "BIDIRECTIONAL"
  local_kafka_cluster {
    id            = data.confluent_kafka_cluster.west.id
    rest_endpoint = data.confluent_kafka_cluster.west.rest_endpoint
  }

  remote_kafka_cluster {
    id                 = data.confluent_kafka_cluster.east.id
    bootstrap_endpoint = data.confluent_kafka_cluster.east.bootstrap_endpoint
  }

  // Make sure confluent_cluster_link.east-to-west is created first to avoid a race condition in the backend
  // if confluent_cluster_link.west-to-east and confluent_cluster_link.east-to-west are created simultaneously,
  // which can cause the cluster link IDs to differ.
  depends_on = [
    confluent_cluster_link.east-to-west
  ]
}

resource "confluent_kafka_mirror_topic" "from-east" {
  source_kafka_topic {
    topic_name = var.east_topic_name
  }
  cluster_link {
    link_name = confluent_cluster_link.east-to-west.link_name
  }
  kafka_cluster {
    id            = data.confluent_kafka_cluster.west.id
    rest_endpoint = data.confluent_kafka_cluster.west.rest_endpoint
  }

  depends_on = [
    confluent_cluster_link.east-to-west,
    confluent_cluster_link.west-to-east,
  ]
}

resource "confluent_kafka_mirror_topic" "from-west" {
  source_kafka_topic {
    topic_name = var.west_topic_name
  }
  cluster_link {
    link_name = confluent_cluster_link.west-to-east.link_name
  }
  kafka_cluster {
    id            = data.confluent_kafka_cluster.east.id
    rest_endpoint = data.confluent_kafka_cluster.east.rest_endpoint
  }

  depends_on = [
    confluent_cluster_link.east-to-west,
    confluent_cluster_link.west-to-east,
  ]
}
