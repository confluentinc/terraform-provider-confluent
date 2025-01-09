terraform {
  required_providers {
    confluent = {
      source = "terraform.confluent.io/confluentinc/confluent"
      # source  = "confluentinc/confluent"
      # version = "2.12.0"
    }
  }
}

provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key
  cloud_api_secret = var.confluent_cloud_api_secret
  endpoint = "https://api.stag.cpdev.cloud"
}

resource "confluent_service_account" "tableflow-sa" {
  display_name = "tableflow-service-account"
  description  = "Service account to manage tableflow api key"
}

resource "confluent_api_key" "tableflow-api-key" {
  display_name = "tableflow-api-key"
  description  = "Tableflow API Key that is owned by 'tableflow-sa' service account lalala"
  owner {
    id          = confluent_service_account.tableflow-sa.id
    api_version = confluent_service_account.tableflow-sa.api_version
    kind        = confluent_service_account.tableflow-sa.kind
  }

  managed_resource {
    id          = "tableflow"
    api_version = "tableflow/v1"
    kind        = "Tableflow"
  }
}

resource "confluent_api_key" "tableflow-api-key2" {
  display_name = "tableflow-api-key2"
  description  = "Tableflow API Key that is owned by 'tableflow-sa' service account lalala"
  owner {
    id          = confluent_service_account.tableflow-sa.id
    api_version = confluent_service_account.tableflow-sa.api_version
    kind        = confluent_service_account.tableflow-sa.kind
  }

  managed_resource {
    id          = "tableflow"
    api_version = "tableflow/v1"
    kind        = "Tableflow"
  }
}

# resource "confluent_api_key" "kafka-api-key" {
#   display_name = "kafka-api-key"
#   description  = "Kafka API Key that is owned by 'tableflow-sa' service account lalala"
#   owner {
#     id          = confluent_service_account.tableflow-sa.id
#     api_version = confluent_service_account.tableflow-sa.api_version
#     kind        = confluent_service_account.tableflow-sa.kind
#   }
#
#   managed_resource {
#     id          = "lkc-ovzrpo"
#     api_version = "cmk/v2"
#     kind        = "Cluster"
#
#     environment {
#       id = "env-3woo02"
#     }
#   }
# }
#
# resource "confluent_api_key" "cloud-api-key" {
#   display_name = "cloud-api-key"
#   description  = "Cloud API Key that is owned by 'tableflow-sa' service account"
#   owner {
#     id          = confluent_service_account.tableflow-sa.id
#     api_version = confluent_service_account.tableflow-sa.api_version
#     kind        = confluent_service_account.tableflow-sa.kind
#   }
# }
