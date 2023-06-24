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
  crn_pattern = confluent_kafka_cluster.source-cluster.rbac_crn
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
    id          = confluent_kafka_cluster.source-cluster.id
    api_version = confluent_kafka_cluster.source-cluster.api_version
    kind        = confluent_kafka_cluster.source-cluster.kind

    environment {
      id = confluent_kafka_cluster.source-cluster.environment.0.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager-source-cluster-admin is created before
  # confluent_api_key.app-manager-source-cluster-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-source-cluster-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-source-cluster-admin,
    module.destination-vpce,
    module.source-vpce,
  ]
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
  crn_pattern = confluent_kafka_cluster.destination-cluster.rbac_crn
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
    id          = confluent_kafka_cluster.destination-cluster.id
    api_version = confluent_kafka_cluster.destination-cluster.api_version
    kind        = confluent_kafka_cluster.destination-cluster.kind

    environment {
      id = confluent_kafka_cluster.destination-cluster.environment.0.id
    }
  }

  # The goal is to ensure that confluent_role_binding.app-manager-destination-cluster-admin is created before
  # confluent_api_key.app-manager-destination-cluster-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-destination-cluster-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-destination-cluster-admin,
    module.destination-vpce,
    module.source-vpce,
  ]
}

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.source-cluster.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster.source-cluster.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-source-cluster-api-key.id
    secret = confluent_api_key.app-manager-source-cluster-api-key.secret
  }

  depends_on = [
    module.destination-vpce,
    module.source-vpce,
  ]
}

resource "confluent_cluster_link" "destination-outbound" {
  link_name = "private_networks_cluster_link"
  source_kafka_cluster {
    id                 = confluent_kafka_cluster.source-cluster.id
    bootstrap_endpoint = confluent_kafka_cluster.source-cluster.bootstrap_endpoint
    credentials {
      key    = confluent_api_key.app-manager-source-cluster-api-key.id
      secret = confluent_api_key.app-manager-source-cluster-api-key.secret
    }
  }

  destination_kafka_cluster {
    id            = confluent_kafka_cluster.destination-cluster.id
    rest_endpoint = confluent_kafka_cluster.destination-cluster.rest_endpoint
    credentials {
      key    = confluent_api_key.app-manager-destination-cluster-api-key.id
      secret = confluent_api_key.app-manager-destination-cluster-api-key.secret
    }
  }

  depends_on = [
    confluent_network_link_endpoint.main,
    module.destination-vpce,
    module.source-vpce,
  ]
}

resource "confluent_kafka_mirror_topic" "test" {
  source_kafka_topic {
    topic_name = confluent_kafka_topic.orders.topic_name
  }
  cluster_link {
    link_name = confluent_cluster_link.destination-outbound.link_name
  }
  kafka_cluster {
    id            = confluent_kafka_cluster.destination-cluster.id
    rest_endpoint = confluent_kafka_cluster.destination-cluster.rest_endpoint
    credentials {
      key    = confluent_api_key.app-manager-destination-cluster-api-key.id
      secret = confluent_api_key.app-manager-destination-cluster-api-key.secret
    }
  }
}
