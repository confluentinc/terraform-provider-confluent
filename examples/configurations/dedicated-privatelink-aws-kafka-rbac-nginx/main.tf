terraform {
  required_version = ">= 0.14.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.17.0"
    }
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.36.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.1"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
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

# Generate random CIDR block for VPC
resource "random_integer" "network_prefix_1" {
  min = 0
  max = 255
}

resource "random_integer" "network_prefix_2" {
  min = 0
  max = 255
}

locals {
  network_addr_prefix = "10.${random_integer.network_prefix_1.result}.${random_integer.network_prefix_2.result}"
  vpc_cidr_block = "${local.network_addr_prefix}.0/24"

  # Calculate subnet CIDRs for 3 availability zones
  subnet_cidrs = [
    "${local.network_addr_prefix}.0/26",   # i=0: 0/26
    "${local.network_addr_prefix}.64/26",  # i=1: 64/26
    "${local.network_addr_prefix}.128/26"  # i=2: 128/26
  ]

  topic_name = "orders"
  topics_confluent_cloud_url = "https://confluent.cloud/environments/${confluent_environment.staging.id}/clusters/${confluent_kafka_cluster.dedicated.id}/topics?topics_filter=showAll"
}

# Get available AZs for the region
data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "main" {
  cidr_block           = local.vpc_cidr_block
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "confluent-privatelink-vpc"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "confluent-privatelink-igw"
  }
}

# Create subnets in 3 availability zones
resource "aws_subnet" "main" {
  count = 3

  vpc_id            = aws_vpc.main.id
  cidr_block        = local.subnet_cidrs[count.index]
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "confluent-privatelink-subnet-${count.index}"
  }
}

# Create route table for public access
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "confluent-privatelink-rt"
  }
}

# Associate route table with first subnet (for EC2 if needed)
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.main[0].id
  route_table_id = aws_route_table.public.id
}

# Generate SSH key pair automatically
resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

# Create key pair for EC2 access
resource "aws_key_pair" "main" {
  key_name   = "confluent-privatelink-key"
  public_key = tls_private_key.main.public_key_openssh
}

# Create security group for EC2
resource "aws_security_group" "ec2" {
  name        = "confluent-privatelink-ec2-sg"
  description = "Security group for EC2 instance to access Confluent Cloud via PrivateLink"
  vpc_id      = aws_vpc.main.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.client_cidr_blocks
    description = "SSH access"
  }

  # HTTPS access
  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
    description = "HTTPS access"
  }

  # Kafka broker access
  ingress {
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
    description = "Kafka broker access"
  }

  # https://docs.confluent.io/cloud/current/networking/aws-pni.html#update-the-security-group-to-block-outbound-traffic
  # SECURITY WARNING: For production deployments, restrict egress to egress = [] to remove the default 0.0.0.0/0 egress rule.
  # This demo intentionally uses 0.0.0.0/0 to allow downloading Confluent CLI, Terraform provider, and related dependencies.
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound traffic"
  }

  tags = {
    Name = "confluent-privatelink-ec2-sg"
  }
}

# Get Amazon Linux 2023 AMI
data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-x86_64"]
  }
}

# Create EC2 instance for testing
resource "aws_instance" "test" {
  ami                         = data.aws_ami.amazon_linux_2023.id
  instance_type               = "t3.micro"
  key_name                    = aws_key_pair.main.key_name
  vpc_security_group_ids      = [aws_security_group.ec2.id]
  subnet_id                   = aws_subnet.main[0].id
  associate_public_ip_address = true

  user_data = <<-EOF
#!/bin/bash
yum update -y
yum install -y wget curl nginx

# Install Confluent CLI
curl -sL --http1.1 https://cnfl.io/cli | sh -s -- -b /usr/local/bin

# Install Terraform
yum-config-manager --add-repo https://rpm.releases.hashicorp.com/AmazonLinux/hashicorp.repo
yum -y install terraform

echo "Confluent Cloud PrivateLink test instance ready" > /home/ec2-user/status.txt
EOF

  tags = {
    Name = "confluent-privatelink-test"
  }
}

# Create Confluent Environment
resource "confluent_environment" "staging" {
  display_name = "Staging"

  stream_governance {
    package = "ESSENTIALS"
  }
}

# Create locals for zone mapping
locals {
  # Create zone ID to subnet ID mapping for PrivateLink
  subnets_to_privatelink = {
    for i, subnet in aws_subnet.main :
    data.aws_availability_zones.available.zone_ids[i] => subnet.id
  }
  dns_domain = confluent_network.private-link.dns_domain
  bootstrap_prefix = split(".", confluent_kafka_cluster.dedicated.bootstrap_endpoint)[0]
}

resource "confluent_network" "private-link" {
  display_name     = "Private Link Network"
  cloud            = "AWS"
  region           = var.region
  connection_types = ["PRIVATELINK"]
  zones            = keys(local.subnets_to_privatelink)
  environment {
    id = confluent_environment.staging.id
  }
  dns_config {
    resolution = "PRIVATE"
  }
}

# Create PrivateLink access
resource "confluent_private_link_access" "aws" {
  display_name = "AWS Private Link Access"
  aws {
    account = var.aws_account_id
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.private-link.id
  }
}

resource "confluent_kafka_cluster" "dedicated" {
  display_name = "inventory"
  availability = "MULTI_ZONE"
  cloud        = confluent_network.private-link.cloud
  region       = confluent_network.private-link.region
  dedicated {
    cku = 2
  }
  environment {
    id = confluent_environment.staging.id
  }
  network {
    id = confluent_network.private-link.id
  }
}

data "confluent_schema_registry_cluster" "essentials" {
  environment {
    id = confluent_environment.staging.id
  }

  depends_on = [
    confluent_kafka_cluster.dedicated
  ]
}

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

  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin,
    confluent_private_link_access.aws,
    aws_vpc_endpoint.privatelink,
    aws_route53_record.privatelink,
    aws_route53_record.privatelink-zonal,
  ]
}

resource "confluent_kafka_topic" "orders" {
  kafka_cluster {
    id = confluent_kafka_cluster.dedicated.id
  }
  topic_name    = local.topic_name
  rest_endpoint = confluent_kafka_cluster.dedicated.rest_endpoint
  credentials {
    key    = confluent_api_key.app-manager-kafka-api-key.id
    secret = confluent_api_key.app-manager-kafka-api-key.secret
  }
}

resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer"
  description  = "Service account to consume from '${local.topic_name}' topic of 'inventory' Kafka cluster"
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

  depends_on = [
    confluent_private_link_access.aws,
    aws_vpc_endpoint.privatelink,
    aws_route53_record.privatelink,
    aws_route53_record.privatelink-zonal,
  ]
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer"
  description  = "Service account to produce to '${local.topic_name}' topic of 'inventory' Kafka cluster"
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

  depends_on = [
    confluent_private_link_access.aws,
    aws_vpc_endpoint.privatelink,
    aws_route53_record.privatelink,
    aws_route53_record.privatelink-zonal,
  ]
}

resource "confluent_role_binding" "app-producer-developer-write" {
  principal   = "User:${confluent_service_account.app-producer.id}"
  role_name   = "DeveloperWrite"
  crn_pattern = "${confluent_kafka_cluster.dedicated.rbac_crn}/kafka=${confluent_kafka_cluster.dedicated.id}/topic=${confluent_kafka_topic.orders.topic_name}"
}

resource "confluent_role_binding" "app-consumer-developer-read-from-topic" {
  principal   = "User:${confluent_service_account.app-consumer.id}"
  role_name   = "DeveloperRead"
  crn_pattern = "${confluent_kafka_cluster.dedicated.rbac_crn}/kafka=${confluent_kafka_cluster.dedicated.id}/topic=${confluent_kafka_topic.orders.topic_name}"
}

resource "confluent_role_binding" "app-consumer-developer-read-from-group" {
  principal = "User:${confluent_service_account.app-consumer.id}"
  role_name = "DeveloperRead"
  crn_pattern = "${confluent_kafka_cluster.dedicated.rbac_crn}/kafka=${confluent_kafka_cluster.dedicated.id}/group=confluent_cli_consumer_*"
}

# AWS PrivateLink infrastructure
data "aws_availability_zone" "privatelink" {
  for_each = local.subnets_to_privatelink
  zone_id  = each.key
}

# Security group for PrivateLink
resource "aws_security_group" "privatelink" {
  name        = "ccloud-privatelink_${local.bootstrap_prefix}_${aws_vpc.main.id}"
  description = "Confluent Cloud Private Link minimal security group for ${confluent_kafka_cluster.dedicated.bootstrap_endpoint} in ${aws_vpc.main.id}"
  vpc_id      = aws_vpc.main.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  ingress {
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = [aws_vpc.main.cidr_block]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_vpc_endpoint" "privatelink" {
  vpc_id            = aws_vpc.main.id
  service_name      = confluent_network.private-link.aws[0].private_link_endpoint_service
  vpc_endpoint_type = "Interface"

  security_group_ids = [
    aws_security_group.privatelink.id,
  ]

  subnet_ids          = [for zone, subnet_id in local.subnets_to_privatelink : subnet_id]
  private_dns_enabled = false

  depends_on = [
    confluent_private_link_access.aws,
  ]
}

resource "aws_route53_zone" "privatelink" {
  name = local.dns_domain

  vpc {
    vpc_id = aws_vpc.main.id
  }
}

resource "aws_route53_record" "privatelink" {
  count   = length(local.subnets_to_privatelink) == 1 ? 0 : 1
  zone_id = aws_route53_zone.privatelink.zone_id
  name    = "*.${aws_route53_zone.privatelink.name}"
  type    = "CNAME"
  ttl     = "60"
  records = [
    aws_vpc_endpoint.privatelink.dns_entry[0]["dns_name"]
  ]
}

locals {
  endpoint_prefix = split(".", aws_vpc_endpoint.privatelink.dns_entry[0]["dns_name"])[0]
}

resource "aws_route53_record" "privatelink-zonal" {
  for_each = local.subnets_to_privatelink

  zone_id = aws_route53_zone.privatelink.zone_id
  name    = length(local.subnets_to_privatelink) == 1 ? "*" : "*.${each.key}"
  type    = "CNAME"
  ttl     = "60"
  records = [
    format("%s-%s%s",
      local.endpoint_prefix,
      data.aws_availability_zone.privatelink[each.key].name,
      replace(aws_vpc_endpoint.privatelink.dns_entry[0]["dns_name"], local.endpoint_prefix, "")
    )
  ]
}
