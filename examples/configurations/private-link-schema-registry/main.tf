terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.17.0"
    }
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.23.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
}

provider "aws" {
  region = var.region
}

data "confluent_organization" "main" {}

data "confluent_environment" "staging" {
  id = var.environment_id
}

locals {
  cloud  = "AWS"
  region = "us-west-2"
}

data "confluent_schema_registry_cluster" "essentials" {
  environment {
    id = data.confluent_environment.staging.id
  }

  depends_on = [
    confluent_kafka_cluster.enterprise
  ]
}

// In Confluent Cloud, an environment is mapped to a Flink catalog, and a Kafka cluster is mapped to a Flink database.
# Update the config to use a cloud provider and region of your choice.
# https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster
resource "confluent_kafka_cluster" "enterprise" {
  display_name = "inventory"
  cloud        = local.cloud
  region       = local.region
  availability = "LOW"
  enterprise {}
  environment {
    id = data.confluent_environment.staging.id
  }
}

resource "confluent_private_link_attachment" "pla" {
  cloud        = "AWS"
  region       = local.region
  display_name = "staging-aws-platt"
  environment {
    id = data.confluent_environment.staging.id
  }
}

module "privatelink" {
  source                   = "./aws-private-link-endpoint"
  vpc_id                   = var.vpc_id
  privatelink_service_name = confluent_private_link_attachment.pla.aws[0].vpc_endpoint_service_name
  dns_domain               = confluent_private_link_attachment.pla.dns_domain
  subnets_to_privatelink   = var.subnets_to_privatelink
}

resource "confluent_private_link_attachment_connection" "plac" {
  display_name = "staging-aws-plattc"
  environment {
    id = data.confluent_environment.staging.id
  }
  aws {
    vpc_endpoint_id = module.privatelink.vpc_endpoint_id
  }

  private_link_attachment {
    id = confluent_private_link_attachment.pla.id
  }
}

// Service account to perform a task within Confluent Cloud, such as executing a Flink statement
resource "confluent_service_account" "statements-runner" {
  display_name = "statements-runner"
  description  = "Service account for running Flink Statements in 'inventory' Kafka cluster"
}

// Service account to set up initial infrastructure, such as creating a schema and a Kafka topic (Flink table)
resource "confluent_service_account" "infrastructure-manager" {
  display_name = "infrastructure-manager"
  description  = "Service account for setting up schemas and Kafka topics (Flink tables)"
}

resource "confluent_role_binding" "infrastructure-manager-environment-admin" {
  principal   = "User:${confluent_service_account.infrastructure-manager.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = replace(data.confluent_environment.staging.resource_name, "stag.cpdev.", "confluent.")
}

resource "confluent_api_key" "infrastructure-manager-kafka-api-key" {
  display_name = "infrastructure-manager-kafka-api-key"
  description  = "Kafka API Key that is owned by 'infrastructure-manager' service account"
  owner {
    id          = confluent_service_account.infrastructure-manager.id
    api_version = confluent_service_account.infrastructure-manager.api_version
    kind        = confluent_service_account.infrastructure-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.enterprise.id
    api_version = confluent_kafka_cluster.enterprise.api_version
    kind        = confluent_kafka_cluster.enterprise.kind

    environment {
      id = data.confluent_environment.staging.id
    }
  }
  # The goal is to ensure that confluent_role_binding.infrastructure-manager-environment-admin is created before
  # confluent_api_key.infrastructure-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic resources.
  # 'depends_on' meta-argument is specified in confluent_api_key.infrastructure-manager-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic resources instead.
  depends_on = [
    confluent_role_binding.infrastructure-manager-environment-admin
  ]
}

resource "confluent_api_key" "infrastructure-manager-schema-registry-api-key" {
  display_name = "infrastructure-manager-schema-registry-api-key"
  description  = "Schema Registry API Key that is owned by 'infrastructure-manager' service account"
  owner {
    id          = confluent_service_account.infrastructure-manager.id
    api_version = confluent_service_account.infrastructure-manager.api_version
    kind        = confluent_service_account.infrastructure-manager.kind
  }
  managed_resource {
    id          = data.confluent_schema_registry_cluster.essentials.id
    api_version = data.confluent_schema_registry_cluster.essentials.api_version
    kind        = data.confluent_schema_registry_cluster.essentials.kind
    environment {
      id = data.confluent_environment.staging.id
    }
  }
  # The goal is to ensure that confluent_role_binding.infrastructure-manager-environment-admin is created before
  # confluent_api_key.infrastructure-manager-schema-registry-api-key is used to create instances of
  # confluent_schema resources.
  # 'depends_on' meta-argument is specified in confluent_api_key.infrastructure-manager-schema-registry-api-key to
  # avoid having multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_schema resources instead.
  depends_on = [
    confluent_role_binding.infrastructure-manager-environment-admin
  ]
}

resource "confluent_role_binding" "statements-runner-environment-admin" {
  principal   = "User:${confluent_service_account.statements-runner.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = replace(data.confluent_environment.staging.resource_name, "stag.cpdev.", "confluent.")
}

// Service account that owns Flink API Key
resource "confluent_service_account" "app-manager" {
  display_name = "app-manager"
  description  = "Service account that has got full access to Flink resources in an environment"
}

// https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#flinkadmin
resource "confluent_role_binding" "app-manager-flink-developer" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "FlinkAdmin"
  crn_pattern = replace(data.confluent_environment.staging.resource_name, "stag.cpdev.", "confluent.")
}

// https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#assigner
// https://docs.confluent.io/cloud/current/flink/operate-and-deploy/flink-rbac.html#submit-long-running-statements
resource "confluent_role_binding" "app-manager-assigner" {
  principal   = "User:${confluent_service_account.app-manager.id}"
  role_name   = "Assigner"
  crn_pattern = replace("${data.confluent_organization.main.resource_name}/service-account=${confluent_service_account.statements-runner.id}", "stag.cpdev.", "confluent.")
}

data "confluent_flink_region" "us-west-2" {
  cloud  = local.cloud
  region = local.region
}

resource "confluent_api_key" "app-manager-flink-api-key" {
  display_name = "app-manager-flink-api-key"
  description  = "Flink API Key that is owned by 'app-manager' service account"
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }
  managed_resource {
    id          = data.confluent_flink_region.us-west-2.id
    api_version = data.confluent_flink_region.us-west-2.api_version
    kind        = data.confluent_flink_region.us-west-2.kind
    environment {
      id = var.environment_id
    }
  }
}

data "confluent_flink_region" "main" {
  cloud  = local.cloud
  region = local.region
}

# https://docs.confluent.io/cloud/current/flink/get-started/quick-start-cloud-console.html#step-1-create-a-af-compute-pool
resource "confluent_flink_compute_pool" "main" {
  display_name = "my-compute-pool"
  cloud        = local.cloud
  region       = local.region
  max_cfu      = 10
  environment {
    id = var.environment_id
  }
  depends_on = [
    confluent_role_binding.statements-runner-environment-admin,
    confluent_role_binding.app-manager-assigner,
    confluent_role_binding.app-manager-flink-developer,
    confluent_api_key.app-manager-flink-api-key,
  ]
}

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.enterprise.id
  }
  topic_name    = "orders_source"
  rest_endpoint = confluent_kafka_cluster.enterprise.rest_endpoint
  credentials {
    key    = confluent_api_key.infrastructure-manager-kafka-api-key.id
    secret = confluent_api_key.infrastructure-manager-kafka-api-key.secret
  }
}

resource "confluent_schema" "order" {
  schema_registry_cluster {
    id = data.confluent_schema_registry_cluster.essentials.id
  }
  rest_endpoint = data.confluent_schema_registry_cluster.essentials.private_rest_endpoint

  # https://developer.confluent.io/learn-kafka/schema-registry/schema-subjects/#topicnamestrategy
  subject_name = "${confluent_kafka_topic.orders.topic_name}-value"
  format       = "AVRO"
  schema       = file("./schemas/avro/order.avsc")
  credentials {
    key    = confluent_api_key.infrastructure-manager-schema-registry-api-key.id
    secret = confluent_api_key.infrastructure-manager-schema-registry-api-key.secret
  }
}
