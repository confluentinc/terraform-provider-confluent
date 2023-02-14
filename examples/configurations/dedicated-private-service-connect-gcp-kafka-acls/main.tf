terraform {
  required_version = ">= 0.14.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "4.18.0"
    }
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.31.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

resource "confluent_environment" "staging" {
  display_name = "Staging"
}

resource "confluent_schema_registry_cluster" "essentials" {
  package = "ESSENTIALS"

  environment {
    id = confluent_environment.staging.id
  }

  region {
    # See https://docs.confluent.io/cloud/current/stream-governance/packages.html#stream-governance-regions
    # Stream Governance and Kafka clusters can be in different regions as well as different cloud providers,
    # but you should to place both in the same cloud and region to restrict the fault isolation boundary.
    id = "sgreg-2"
  }
}

resource "confluent_network" "private-service-connect" {
  display_name     = "Private Service Connect Network"
  cloud            = "GCP"
  region           = var.region
  connection_types = ["PRIVATELINK"]
  zones            = keys(var.subnet_name_by_zone)
  environment {
    id = confluent_environment.staging.id
  }
}

resource "confluent_private_link_access" "gcp" {
  display_name = "GCP Private Service Connect"
  gcp {
    project = var.customer_project_id
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.private-service-connect.id
  }
}

resource "confluent_kafka_cluster" "dedicated" {
  display_name = "inventory"
  availability = "MULTI_ZONE"
  cloud        = confluent_network.private-service-connect.cloud
  region       = confluent_network.private-service-connect.region
  dedicated {
    cku = 2
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.private-service-connect.id
  }
}

// 'app-manager' service account is required in this configuration to create 'orders' topic and grant ACLs
// to 'app-producer' and 'app-consumer' service accounts.
resource "confluent_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "app-manager-kafka-cluster-admin" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "CloudClusterAdmin"
  crn_pattern = confluent_kafka_cluster.dedicated.rbac_crn
}

resource "confluent_api_key" "app-manager-kafka-api-key" {
  display_name = "app-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that
  # 1. confluent_role_binding.app-manager-kafka-cluster-admin is created before
  # confluent_api_key.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.
  # 2. Kafka connectivity through GCP Private Service Connect is setup.
  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin,

    confluent_private_link_access.gcp,
    google_compute_forwarding_rule.psc_endpoint_ilb,
    google_dns_record_set.psc_endpoint_rs,
    google_dns_record_set.psc_endpoint_zonal_rs,
    google_compute_firewall.allow-https-kafka,
  ]
}

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  topic_name    = "orders"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-consumer-kafka-api-key" {
  display_name = "app-consumer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-consumer' service account"
  owner {
    id          = confluent_service_account.app-consumer.id
    api_version = confluent_service_account.app-consumer.api_version
    kind        = confluent_service_account.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that Kafka connectivity through AWS PrivateLink is setup.
  depends_on = [
    confluent_private_link_access.gcp,
    google_compute_forwarding_rule.psc_endpoint_ilb,
    google_dns_record_set.psc_endpoint_rs,
    google_dns_record_set.psc_endpoint_zonal_rs,
    google_compute_firewall.allow-https-kafka,
  ]
}

resource "confluent_kafka_acl" "app-producer-write-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-producer.id}"
  host          = "*"
  operation     = "WRITE"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to 'orders' topic of 'inventory' Kafka cluster"
}

resource "confluent_api_key" "app-producer-kafka-api-key" {
  display_name = "app-producer-kafka-api-key"
  description  = "Kafka API Key that is owned by 'app-producer' service account"
  owner {
    id          = confluent_service_account.app-producer.id
    api_version = confluent_service_account.app-producer.api_version
    kind        = confluent_service_account.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.dedicated.id
    api_version = confluent_kafka_cluster.dedicated.api_version
    kind        = confluent_kafka_cluster.dedicated.kind

    environment {
      id = confluent_environment.staging.id
    }
  }

  # The goal is to ensure that Kafka connectivity through GCP Private Service Connect is setup.
  depends_on = [
    confluent_private_link_access.gcp,
    google_compute_forwarding_rule.psc_endpoint_ilb,
    google_dns_record_set.psc_endpoint_rs,
    google_dns_record_set.psc_endpoint_zonal_rs,
    google_compute_firewall.allow-https-kafka,
  ]
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
// confluent_kafka_acl.app-consumer-read-on-topic, confluent_kafka_acl.app-consumer-read-on-group.
// https://docs.confluent.io/platform/current/kafka/authorization.html#using-acls
resource "confluent_kafka_acl" "app-consumer-read-on-topic" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "TOPIC"
  resource_name = confluent_kafka_topic.orders.topic_name
  pattern_type  = "LITERAL"
  principal     = "User:${confluent_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_kafka_acl" "app-consumer-read-on-group" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  resource_type = "GROUP"
  // The existing values of resource_name, pattern_type attributes are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_consume.html
  // Update the values of resource_name, pattern_type attributes to match your target consumer group ID.
  // https://docs.confluent.io/platform/current/kafka/authorization.html#prefixed-acls
  resource_name = "confluent_cli_consumer_"
  pattern_type  = "PREFIXED"
  principal     = "User:${confluent_service_account.app-consumer.id}"
  host          = "*"
  operation     = "READ"
  permission    = "ALLOW"
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

# Set GOOGLE_APPLICATION_CREDENTIALS environment variable to a path to a key file
# for Google TF Provider to work: https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#adding-credentials
provider "google" {
  project = var.customer_project_id
  region  = var.region
}

locals {
  hosted_zone = length(regexall(".glb", confluent_kafka_cluster.dedicated.bootstrap_endpoint)) > 0 ? replace(regex("^[^.]+-([0-9a-zA-Z]+[.].*):[0-9]+$", confluent_kafka_cluster.dedicated.bootstrap_endpoint)[0], "glb.", "") : regex("[.]([0-9a-zA-Z]+[.].*):[0-9]+$", confluent_kafka_cluster.dedicated.bootstrap_endpoint)[0]
  network_id  = regex("^([^.]+)[.].*", local.hosted_zone)[0]
}

data "google_compute_network" "psc_endpoint_network" {
  name = var.customer_vpc_network
}

data "google_compute_subnetwork" "psc_endpoint_subnetwork" {
  name = var.customer_subnetwork_name
}

resource "google_compute_address" "psc_endpoint_ip" {
  for_each = var.subnet_name_by_zone

  name         = "ccloud-endpoint-ip-${local.network_id}-${each.key}"
  subnetwork   = var.customer_subnetwork_name
  address_type = "INTERNAL"
}

# Private Service Connect endpoint
resource "google_compute_forwarding_rule" "psc_endpoint_ilb" {
  for_each = var.subnet_name_by_zone

  name = "ccloud-endpoint-${local.network_id}-${each.key}"

  target                = lookup(confluent_network.private-service-connect.gcp[0].private_service_connect_service_attachments, each.key, "\n\nerror: ${each.key} subnet is missing from CCN's Private Service Connect service attachments")
  load_balancing_scheme = "" # need to override EXTERNAL default when target is a service attachment
  network               = var.customer_vpc_network
  ip_address            = google_compute_address.psc_endpoint_ip[each.key].id
}

# Private hosted zone for Private Service Connect endpoints
resource "google_dns_managed_zone" "psc_endpoint_hz" {
  name     = "ccloud-endpoint-zone-${local.network_id}"
  dns_name = "${local.hosted_zone}."

  visibility = "private"

  private_visibility_config {
    networks {
      network_url = data.google_compute_network.psc_endpoint_network.id
    }
  }
}

resource "google_dns_record_set" "psc_endpoint_rs" {
  name = "*.${google_dns_managed_zone.psc_endpoint_hz.dns_name}"
  type = "A"
  ttl  = 60

  managed_zone = google_dns_managed_zone.psc_endpoint_hz.name
  rrdatas = [
    for zone, _ in var.subnet_name_by_zone : google_compute_address.psc_endpoint_ip[zone].address
  ]
}

resource "google_dns_record_set" "psc_endpoint_zonal_rs" {
  for_each = var.subnet_name_by_zone

  name = "*.${each.key}.${google_dns_managed_zone.psc_endpoint_hz.dns_name}"
  type = "A"
  ttl  = 60

  managed_zone = google_dns_managed_zone.psc_endpoint_hz.name
  rrdatas      = [google_compute_address.psc_endpoint_ip[each.key].address]
}

resource "google_compute_firewall" "allow-https-kafka" {
  name    = "ccloud-endpoint-firewall-${local.network_id}"
  network = data.google_compute_network.psc_endpoint_network.id

  allow {
    protocol = "tcp"
    ports    = ["80", "443", "9092"]
  }

  direction          = "EGRESS"
  destination_ranges = [data.google_compute_subnetwork.psc_endpoint_subnetwork.ip_cidr_range]
}
