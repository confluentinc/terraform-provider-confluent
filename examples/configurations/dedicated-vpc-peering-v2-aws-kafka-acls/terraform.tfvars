# Cross-region access to Confluent Cloud is not supported when VPC peering is enabled with Google Cloud. Your AWS VPC and Confluent Cloud must be in the same region.
# The region of your VPC that you want to connect to Confluent Cloud Cluster
# Cross-region AWS PrivateLink connections are not supported yet.
region = "us-east-1"

# The region of the AWS peer VPC.
customer_region = "us-east-1"

# The Confluent Cloud Network Availability Zones Metadata.
zones_info = [
  { zone_id : "use1-az1", cidr : "192.168.2.0/27" },
  { zone_id : "use1-az2", cidr : "192.168.2.32/27" },
  { zone_id : "use1-az3", cidr : "192.168.2.64/27" }
]

# The AWS Account ID of the peer VPC owner.
# You can find your AWS Account ID here (https://console.aws.amazon.com/billing/home?#/account) under My Account section of the AWS Management Console. Must be a 12 character string.
aws_account_id = "012345678901"

# The AWS VPC ID of the peer VPC that you're peering with Confluent Cloud.
# You can find your AWS VPC ID here (https://console.aws.amazon.com/vpc/) under Your VPCs section of the AWS Management Console. Must start with `vpc-`.
vpc_id = "vpc-abcdef0123456789a"

# Add credentials and other settings to $HOME/.aws/config
# for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files

# Requirements of VPC Peering on AWS
# https://docs.confluent.io/cloud/current/networking/peering/aws-peering.html#vpc-peering-on-aws
