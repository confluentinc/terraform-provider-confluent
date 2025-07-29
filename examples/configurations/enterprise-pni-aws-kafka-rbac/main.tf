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

# Generate random CIDR block for VPC (equivalent to network_addr_prefix="10.$((RANDOM % 256)).$((RANDOM % 256))")
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

  # Calculate subnet CIDRs (equivalent to subnet_cidr="$network_addr_prefix.$((i * 64))/26")
  subnet_cidrs = [
    "${local.network_addr_prefix}.0/26", # i=0: 0/26
    "${local.network_addr_prefix}.64/26", # i=1: 64/26
    "${local.network_addr_prefix}.128/26" # i=2: 128/26
  ]

  # Calculate base IPs (equivalent to base_ips+=("$network_addr_prefix.$((i * 64 + 10))"))
  base_ips = [
    "${local.network_addr_prefix}.10", # i=0: 0 + 10 = 10
    "${local.network_addr_prefix}.74", # i=1: 64 + 10 = 74
    "${local.network_addr_prefix}.138" # i=2: 128 + 10 = 138
  ]
}

# Create VPC (equivalent to aws ec2 create-vpc)
resource "aws_vpc" "main" {
  cidr_block           = local.vpc_cidr_block
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "enterprise-pni-aws-kafka-rbac"
  }
}

# Create single security group for demo (both EC2 and ENIs)
resource "aws_security_group" "main" {
  name        = "pni-demo-sg-${var.environment_id}"
  description = "Demo security group for PNI test (EC2 + ENIs)"
  vpc_id      = aws_vpc.main.id

  # SSH access for EC2
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
    cidr_blocks = concat(var.client_cidr_blocks, [aws_vpc.main.cidr_block])
    description = "HTTPS access"
  }

  # Kafka broker access for ENIs
  ingress {
    from_port   = 9092
    to_port     = 9092
    protocol    = "tcp"
    cidr_blocks = concat(var.client_cidr_blocks, [aws_vpc.main.cidr_block])
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
    Name = "enterprise-pni-aws-kafka-rbac"
  }
}

# Generate SSH key pair automatically
resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 2048
}

# Create key pair for EC2 access
resource "aws_key_pair" "main" {
  key_name   = "pni-test-key-${var.environment_id}"
  public_key = tls_private_key.main.public_key_openssh
}

# Create Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "enterprise-pni-aws-kafka-rbac"
  }
}

# Create route table for public subnet
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Name = "enterprise-pni-aws-kafka-rbac"
  }
}

# Associate route table with first subnet (for EC2)
resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.main[0].id
  route_table_id = aws_route_table.public.id
}

data "aws_ami" "amazon_linux_2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-2023.*-x86_64"]
  }
}

locals {
  pni_kafka_rest_endpoint = [for endpoint in confluent_kafka_cluster.enterprise.endpoints : endpoint.rest_endpoint if endpoint.access_point_id == confluent_access_point.aws.id][0]
  pni_bootstrap_endpoint = [for endpoint in confluent_kafka_cluster.enterprise.endpoints : endpoint.bootstrap_endpoint if endpoint.access_point_id == confluent_access_point.aws.id][0]
}

# Create EC2 instance
resource "aws_instance" "test" {
  ami                         = data.aws_ami.amazon_linux_2023.id
  instance_type               = "t2.micro"
  key_name                    = aws_key_pair.main.key_name
  vpc_security_group_ids      = [aws_security_group.main.id]
  subnet_id                   = aws_subnet.main[0].id
  associate_public_ip_address = true

  user_data = <<-EOF
#!/bin/bash
set -e

yum update -y
# Install nginx and stream module (Amazon Linux 2023 specific)
yum install -y wget yum-utils nginx nginx-mod-stream bind-utils

# START of setting up https://docs.confluent.io/cloud/current/networking/ccloud-console-access.html#configure-a-proxy
BOOTSTRAP_HOST="${local.pni_bootstrap_endpoint}"

echo "Setting up NGINX proxy for Confluent Cloud PNI" >> /var/log/user-data.log
echo "Bootstrap host: $BOOTSTRAP_HOST" >> /var/log/user-data.log

# Step 3: Test NGINX configuration (before we modify it)
echo "Testing initial NGINX configuration..." >> /var/log/user-data.log
nginx -t >> /var/log/user-data.log 2>&1

# Step 4: Check if ngx_stream_module.so exists and set MODULE_PATH
echo "Checking for stream module..." >> /var/log/user-data.log
if [ -f /usr/lib64/nginx/modules/ngx_stream_module.so ]; then
  MODULE_PATH="/usr/lib64/nginx/modules/ngx_stream_module.so"
  echo "Found stream module at: $MODULE_PATH" >> /var/log/user-data.log
elif [ -f /usr/lib/nginx/modules/ngx_stream_module.so ]; then
  MODULE_PATH="/usr/lib/nginx/modules/ngx_stream_module.so"
  echo "Found stream module at: $MODULE_PATH" >> /var/log/user-data.log
else
  echo "ERROR: ngx_stream_module.so not found!" >> /var/log/user-data.log
  exit 1
fi

# Step 5: Use AWS resolver directly (we know it works on EC2)
RESOLVER="169.254.169.253"
echo "Using AWS resolver: $RESOLVER" >> /var/log/user-data.log

# Step 6: Update NGINX configuration
cat > /etc/nginx/nginx.conf <<NGINXCONF
load_module $MODULE_PATH;

events {}
stream {
  map \$ssl_preread_server_name \$targetBackend {
    default \$ssl_preread_server_name;
  }

  server {
    listen 9092;
    proxy_connect_timeout 1s;
    proxy_timeout 7200s;
    resolver $RESOLVER;
    proxy_pass \$targetBackend:9092;
    ssl_preread on;
  }

  server {
    listen 443;
    proxy_connect_timeout 1s;
    proxy_timeout 7200s;
    resolver $RESOLVER;
    proxy_pass \$targetBackend:443;
    ssl_preread on;
  }

  log_format stream_routing '[\$time_local] remote address \$remote_addr '
                            'with SNI name "\$ssl_preread_server_name" '
                            'proxied to "\$upstream_addr" '
                            '\$protocol \$status \$bytes_sent \$bytes_received '
                            '\$session_time';
  access_log /var/log/nginx/stream-access.log stream_routing;
}
NGINXCONF

# Step 7: Re-test NGINX configuration
echo "Testing NGINX configuration after update..." >> /var/log/user-data.log
if nginx -t >> /var/log/user-data.log 2>&1; then
  echo "NGINX configuration test passed" >> /var/log/user-data.log
else
  echo "NGINX configuration test failed:" >> /var/log/user-data.log
  nginx -t >> /var/log/user-data.log 2>&1
  exit 1
fi

# Step 8: Restart NGINX
echo "Restarting NGINX..." >> /var/log/user-data.log
systemctl restart nginx

# Step 9: Verify NGINX is running
echo "Verifying NGINX status..." >> /var/log/user-data.log
if systemctl is-active --quiet nginx; then
  echo "NGINX is running successfully" >> /var/log/user-data.log
  systemctl status nginx >> /var/log/user-data.log 2>&1
else
  echo "NGINX failed to start:" >> /var/log/user-data.log
  systemctl status nginx >> /var/log/user-data.log 2>&1
  # Check error logs as suggested in Confluent docs
  echo "NGINX error log:" >> /var/log/user-data.log
  tail -20 /var/log/nginx/error.log >> /var/log/user-data.log 2>&1
  exit 1
fi

# Enable NGINX to start on boot
systemctl enable nginx

# Install Confluent CLI
echo "Installing Confluent CLI..." >> /var/log/user-data.log
mkdir -p /usr/local/bin
curl -sL --http1.1 https://cnfl.io/cli | sh -s -- -b /usr/local/bin

# Install Terraform
echo "Installing Terraform..." >> /var/log/user-data.log
yum-config-manager --add-repo https://rpm.releases.hashicorp.com/AmazonLinux/hashicorp.repo
yum -y install terraform

# Verify installations
if /usr/local/bin/confluent version >> /var/log/user-data.log 2>&1; then
  echo "Confluent CLI installed successfully" >> /var/log/user-data.log
else
  echo "Confluent CLI installation failed" >> /var/log/user-data.log
fi

if terraform version >> /var/log/user-data.log 2>&1; then
  echo "Terraform installed successfully" >> /var/log/user-data.log
else
  echo "Terraform installation failed" >> /var/log/user-data.log
fi

echo "Proxy setup completed successfully!" >> /var/log/user-data.log
echo "You can now test with: nslookup $BOOTSTRAP_HOST $RESOLVER" >> /var/log/user-data.log
EOF

  tags = {
    Name = "enterprise-pni-aws-kafka-rbac"
  }
}

# Create subnets (equivalent to aws ec2 create-subnet loop)
resource "aws_subnet" "main" {
  count = 3

  vpc_id               = aws_vpc.main.id
  cidr_block           = local.subnet_cidrs[count.index]
  availability_zone_id = var.availability_zone_ids[count.index]

  tags = {
    Name = "subnet-${count.index}"
  }
}

# Create ENIs (equivalent to enis_create.sh script)
# For each subnet (0, 1, 2), create num_eni_per_subnet ENIs
resource "aws_network_interface" "main" {
  count = 3 * var.num_eni_per_subnet

  subnet_id = aws_subnet.main[floor(count.index / var.num_eni_per_subnet)].id
  security_groups = [aws_security_group.main.id]

  # Calculate private IP: base_ip + (j+1) where j is the ENI number within subnet
  # floor(count.index / var.num_eni_per_subnet) gives subnet index (0, 1, 2)
  # count.index % var.num_eni_per_subnet gives ENI index within subnet (0, 1, ...)
  private_ips = [
    cidrhost(
      aws_subnet.main[floor(count.index / var.num_eni_per_subnet)].cidr_block,
      10 + (count.index % var.num_eni_per_subnet) + 1
    )
  ]

  description = "Confluent PNI-sub-${floor(count.index / var.num_eni_per_subnet)}-eni-${(count.index % var.num_eni_per_subnet) + 1}"

  tags = {
    Name = "Confluent-PNI-sub-${floor(count.index / var.num_eni_per_subnet)}-eni-${(count.index % var.num_eni_per_subnet) + 1}"
  }

  depends_on = [
    confluent_gateway.main
  ]
}

# Create network interface permissions (equivalent to aws ec2 create-network-interface-permission)
resource "aws_network_interface_permission" "main" {
  count = length(aws_network_interface.main)

  network_interface_id = aws_network_interface.main[count.index].id
  permission           = "INSTANCE-ATTACH"
  aws_account_id       = confluent_gateway.main.aws_private_network_interface_gateway[0].account
}

data "confluent_environment" "staging" {
  id = var.environment_id
}

resource "confluent_gateway" "main" {
  display_name = "my_gateway"
  environment {
    id = data.confluent_environment.staging.id
  }
  aws_private_network_interface_gateway {
    region = var.region
    zones  = var.availability_zone_ids
  }
}

resource "confluent_access_point" "aws" {
  display_name = "my_access_point"
  environment {
    id = data.confluent_environment.staging.id
  }
  gateway {
    id = confluent_gateway.main.id
  }
  aws_private_network_interface {
    network_interfaces = aws_network_interface.main[*].id
    account            = var.aws_account_id
  }

  depends_on = [
    aws_network_interface_permission.main
  ]
}

# Run on EC2 instance
resource "confluent_kafka_cluster" "enterprise" {
  display_name = "inventory"
  availability = "HIGH"
  cloud        = "AWS"
  region       = var.region
  enterprise {}
  environment {
    id = data.confluent_environment.staging.id
  }

  depends_on = [
    aws_network_interface_permission.main
  ]
}

resource "confluent_service_account" "app-manager" {
  display_name = "app-manager-${confluent_kafka_cluster.enterprise.id}"
  description  = "Service account to manage 'inventory' Kafka cluster"
}

resource "confluent_role_binding" "app-manager-kafka-cluster-admin" {
  principal = "User:${confluent_service_account.app-manager.id}"
  role_name = "CloudClusterAdmin"
  crn_pattern = replace(confluent_kafka_cluster.enterprise.rbac_crn, "stag.cpdev.cloud", "confluent.cloud")
}

resource "confluent_api_key" "app-manager-kafka-api-key" {
  display_name           = "${confluent_service_account.app-manager.display_name}-kafka-api-key"
  description            = "Kafka API Key that is owned by 'app-manager' service account"
  disable_wait_for_ready = true
  owner {
    id          = confluent_service_account.app-manager.id
    api_version = confluent_service_account.app-manager.api_version
    kind        = confluent_service_account.app-manager.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.enterprise.id
    api_version = confluent_kafka_cluster.enterprise.api_version
    kind        = confluent_kafka_cluster.enterprise.kind

    environment {
      id = data.confluent_environment.staging.id
    }
  }
  # The goal is to ensure that confluent_role_binding.app-manager-kafka-cluster-admin is created before
  # confluent_api_key.app-manager-kafka-api-key is used to create instances of
  # confluent_kafka_topic, confluent_kafka_acl resources.

  # 'depends_on' meta-argument is specified in confluent_api_key.app-manager-kafka-api-key to avoid having
  # multiple copies of this definition in the configuration which would happen if we specify it in
  # confluent_kafka_topic, confluent_kafka_acl resources instead.
  depends_on = [
    confluent_role_binding.app-manager-kafka-cluster-admin
  ]
}

locals {
  topic_name = "orders"
}

resource "confluent_service_account" "app-consumer" {
  display_name = "app-consumer-${confluent_kafka_cluster.enterprise.id}"
  description  = "Service account to consume from '${local.topic_name}' topic of ${confluent_kafka_cluster.enterprise.id} Kafka cluster"
}

resource "confluent_api_key" "app-consumer-kafka-api-key" {
  display_name           = "${confluent_service_account.app-consumer.display_name}-kafka-api-key"
  description            = "Kafka API Key that is owned by ${confluent_service_account.app-consumer.display_name} service account"
  disable_wait_for_ready = true
  owner {
    id          = confluent_service_account.app-consumer.id
    api_version = confluent_service_account.app-consumer.api_version
    kind        = confluent_service_account.app-consumer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.enterprise.id
    api_version = confluent_kafka_cluster.enterprise.api_version
    kind        = confluent_kafka_cluster.enterprise.kind

    environment {
      id = data.confluent_environment.staging.id
    }
  }
}

resource "confluent_role_binding" "app-producer-developer-write" {
  principal = "User:${confluent_service_account.app-producer.id}"
  role_name = "DeveloperWrite"
  crn_pattern = replace("${confluent_kafka_cluster.enterprise.rbac_crn}/kafka=${confluent_kafka_cluster.enterprise.id}/topic=${local.topic_name}", "stag.cpdev.cloud", "confluent.cloud")
}

resource "confluent_service_account" "app-producer" {
  display_name = "app-producer-${confluent_kafka_cluster.enterprise.id}"
  description  = "Service account to produce to '${local.topic_name}' topic of ${confluent_kafka_cluster.enterprise.id} Kafka cluster"
}

resource "confluent_api_key" "app-producer-kafka-api-key" {
  display_name           = "${confluent_service_account.app-producer.display_name}-kafka-api-key"
  description            = "Kafka API Key that is owned by ${confluent_service_account.app-producer.display_name} service account"
  disable_wait_for_ready = true
  owner {
    id          = confluent_service_account.app-producer.id
    api_version = confluent_service_account.app-producer.api_version
    kind        = confluent_service_account.app-producer.kind
  }

  managed_resource {
    id          = confluent_kafka_cluster.enterprise.id
    api_version = confluent_kafka_cluster.enterprise.api_version
    kind        = confluent_kafka_cluster.enterprise.kind

    environment {
      id = data.confluent_environment.staging.id
    }
  }
}

// Note that in order to consume from a topic, the principal of the consumer ('app-consumer' service account)
// needs to be authorized to perform 'READ' operation on both Topic and Group resources:
resource "confluent_role_binding" "app-consumer-developer-read-from-topic" {
  principal = "User:${confluent_service_account.app-consumer.id}"
  role_name = "DeveloperRead"
  crn_pattern = replace("${confluent_kafka_cluster.enterprise.rbac_crn}/kafka=${confluent_kafka_cluster.enterprise.id}/topic=${local.topic_name}", "stag.cpdev.cloud", "confluent.cloud")
}

resource "confluent_role_binding" "app-consumer-developer-read-from-group" {
  principal = "User:${confluent_service_account.app-consumer.id}"
  role_name = "DeveloperRead"
  // The existing value of crn_pattern's suffix (group=confluent_cli_consumer_*) are set up to match Confluent CLI's default consumer group ID ("confluent_cli_consumer_<uuid>").
  // https://docs.confluent.io/confluent-cli/current/command-reference/kafka/topic/confluent_kafka_topic_consume.html
  // Update it to match your target consumer group ID.
  crn_pattern = replace("${confluent_kafka_cluster.enterprise.rbac_crn}/kafka=${confluent_kafka_cluster.enterprise.id}/group=confluent_cli_consumer_*", "stag.cpdev.cloud", "confluent.cloud")
}
