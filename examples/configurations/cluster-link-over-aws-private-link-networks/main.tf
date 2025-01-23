terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.13.0"
    }

    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.17.0"
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

resource "confluent_environment" "main" {
  display_name = "Staging"
}

resource "confluent_network" "source-network" {
  display_name     = "Network Link Service Network"
  cloud            = "AWS"
  region           = var.region
  connection_types = ["PRIVATELINK"]
  zones            = ["usw2-az1", "usw2-az2", "usw2-az3"]
  environment {
    id = confluent_environment.main.id
  }
}

resource "confluent_network" "destination-network" {
  display_name     = "Network Link Endpoint Network"
  cloud            = "AWS"
  region           = var.region
  connection_types = ["PRIVATELINK"]
  zones            = ["usw2-az1", "usw2-az2", "usw2-az3"]
  environment {
    id = confluent_environment.main.id
  }
}

resource "confluent_private_link_access" "source-access" {
  display_name = "NLS Network Private Link Access"
  aws {
    account = var.aws_account_id
  }
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.source-network.id
  }
}

resource "confluent_private_link_access" "destination-access" {
  display_name = "NLE Network Private Link Access"
  aws {
    account = var.aws_account_id
  }
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.destination-network.id
  }
}

resource "confluent_kafka_cluster" "destination-cluster" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = confluent_network.destination-network.cloud
  region       = confluent_network.destination-network.region
  dedicated {
    cku = 1
  }
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.destination-network.id
  }

  depends_on = [
    confluent_private_link_access.destination-access
  ]
}

resource "confluent_kafka_cluster" "source-cluster" {
  display_name = "inventory"
  availability = "SINGLE_ZONE"
  cloud        = confluent_network.source-network.cloud
  region       = confluent_network.source-network.region
  dedicated {
    cku = 1
  }
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.source-network.id
  }

  depends_on = [
    confluent_private_link_access.source-access
  ]
}

resource "confluent_network_link_service" "main" {
  display_name = "network_link_service"
  description  = "Network Link Service"
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.source-network.id
  }
  accept {
    networks = [confluent_network.destination-network.id]
  }
}

resource "confluent_network_link_endpoint" "main" {
  display_name = "network_link_endpoint"
  description  = "Network Link Endpoint"
  environment {
    id = confluent_environment.main.id
  }
  network {
    id = confluent_network.destination-network.id
  }
  network_link_service {
    id = confluent_network_link_service.main.id
  }

  depends_on = [
    confluent_kafka_cluster.source-cluster,
    confluent_kafka_cluster.destination-cluster
  ]
}

module "destination-vpce" {
  source                   = "./aws-privatelink-endpoint"
  vpc_id                   = var.vpc_id
  privatelink_service_name = confluent_network.destination-network.aws[0].private_link_endpoint_service
  dns_domain               = confluent_network.destination-network.dns_domain
  subnets_to_privatelink   = var.subnets_to_privatelink

  depends_on = [
    confluent_private_link_access.destination-access,
  ]
}

module "source-vpce" {
  source                   = "./aws-privatelink-endpoint"
  vpc_id                   = var.vpc_id
  privatelink_service_name = confluent_network.source-network.aws[0].private_link_endpoint_service
  dns_domain               = confluent_network.source-network.dns_domain
  subnets_to_privatelink   = var.subnets_to_privatelink
  depends_on = [
    confluent_private_link_access.source-access,
  ]
}
