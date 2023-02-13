terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.30.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

data "confluent_kafka_cluster" "source" {
  id = var.source_kafka_cluster_id
  environment {
    id = var.source_kafka_cluster_environment_id
  }
}

resource "confluent_service_account" "app-manager-source-cluster" {
  display_name = "app-manager-source-cluster"
  description  = "Service account to manage source Kafka cluster"
}

// See
// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/security-cloud.html#rbac-roles-and-kafka-acls-summary
// and
// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/security-cloud.html#ccloud-rbac-roles for more details.
resource "confluent_role_binding" "app-manager-source-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager-source-cluster.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = data.confluent_kafka_cluster.source.rbac_crn
}

resource "confluent_api_key" "app-manager-source-cluster-api-key" {
  display_name = "app-manager-source-cluster-api-key"
  description  = "Kafka API Key that is owned by 'app-manager-source-cluster' service account"
  owner {
    id          = confluent_service_account.app-manager-source-cluster.id
    api_version = confluent_service_account.app-manager-source-cluster.api_version
    kind        = confluent_service_account.app-manager-source-cluster.kind
  }

  managed_resource {
    id          = data.confluent_kafka_cluster.source.id
    api_version = data.confluent_kafka_cluster.source.api_version
    kind        = data.confluent_kafka_cluster.source.kind

    environment {
      id = data.confluent_kafka_cluster.source.environment.0.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager-source-cluster-admin is created before
  # confluent_api_key.app-manager-source-cluster-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-source-cluster-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-source-cluster-admin
  ]
}

data "confluent_kafka_cluster" "destination" {
  id = var.destination_kafka_cluster_id
  environment {
    id = var.destination_kafka_cluster_environment_id
  }
}

resource "confluent_service_account" "app-manager-destination-cluster" {
  display_name = "app-manager-destination-cluster"
  description  = "Service account to manage destination Kafka cluster"
}

// See
// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/security-cloud.html#rbac-roles-and-kafka-acls-summary
// and
// https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/security-cloud.html#ccloud-rbac-roles for more details.
resource "confluent_role_binding" "app-manager-destination-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager-destination-cluster.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = data.confluent_kafka_cluster.destination.rbac_crn
}

resource "confluent_api_key" "app-manager-destination-cluster-api-key" {
  display_name = "app-manager-destination-cluster-api-key"
  description  = "Kafka API Key that is owned by 'app-manager-destination-cluster' service account"
  owner {
    id          = confluent_service_account.app-manager-destination-cluster.id
    api_version = confluent_service_account.app-manager-destination-cluster.api_version
    kind        = confluent_service_account.app-manager-destination-cluster.kind
  }

  managed_resource {
    id          = data.confluent_kafka_cluster.destination.id
    api_version = data.confluent_kafka_cluster.destination.api_version
    kind        = data.confluent_kafka_cluster.destination.kind

    environment {
      id = data.confluent_kafka_cluster.destination.environment.0.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager-destination-cluster-admin is created before
  # confluent_api_key.app-manager-destination-cluster-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-destination-cluster-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-destination-cluster-admin
  ]
}

resource "confluent_cluster_link" "source-outbound" {
  link_name       = var.cluster_link_name
  link_mode       = "SOURCE"
  connection_mode = "OUTBOUND"
  source_kafka_cluster {
    id            = data.confluent_kafka_cluster.source.id
    rest_endpoint = data.confluent_kafka_cluster.source.rest_endpoint
    credentials {
      key    = confluent_api_key.app-manager-source-cluster-api-key.id
      secret = confluent_api_key.app-manager-source-cluster-api-key.secret
    }
  }

  destination_kafka_cluster {
    id                 = data.confluent_kafka_cluster.destination.id
    bootstrap_endpoint = data.confluent_kafka_cluster.destination.bootstrap_endpoint
    credentials {
      key    = confluent_api_key.app-manager-destination-cluster-api-key.id
      secret = confluent_api_key.app-manager-destination-cluster-api-key.secret
    }
  }

  depends_on = [
    confluent_cluster_link.destination-inbound
  ]
}

resource "confluent_cluster_link" "destination-inbound" {
  link_name       = var.cluster_link_name
  link_mode       = "DESTINATION"
  connection_mode = "INBOUND"
  destination_kafka_cluster {
    id            = data.confluent_kafka_cluster.destination.id
    rest_endpoint = data.confluent_kafka_cluster.destination.rest_endpoint
    credentials {
      key    = confluent_api_key.app-manager-destination-cluster-api-key.id
      secret = confluent_api_key.app-manager-destination-cluster-api-key.secret
    }
  }

  source_kafka_cluster {
    id                 = data.confluent_kafka_cluster.source.id
    bootstrap_endpoint = data.confluent_kafka_cluster.source.bootstrap_endpoint
  }
}

resource "confluent_kafka_mirror_topic" "test" {
  source_kafka_topic {
    topic_name = var.source_topic_name
  }
  cluster_link {
    link_name = confluent_cluster_link.source-outbound.link_name
  }
  kafka_cluster {
    id            = data.confluent_kafka_cluster.destination.id
    rest_endpoint = data.confluent_kafka_cluster.destination.rest_endpoint
    credentials {
      key    = confluent_api_key.app-manager-destination-cluster-api-key.id
      secret = confluent_api_key.app-manager-destination-cluster-api-key.secret
    }
  }
}
