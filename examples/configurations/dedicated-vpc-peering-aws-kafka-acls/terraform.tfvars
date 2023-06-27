# Cross-region access to Confluent Cloud is not supported when VPC peering is enabled with Google Cloud. Your AWS VPC and Confluent Cloud must be in the same region.
# The region of your VPC that you want to connect to Confluent Cloud Cluster
# Cross-region AWS PrivateLink connections are not supported yet.
region = "us-east-1"

# The region of the AWS peer VPC.
customer_region = "us-east-1"

# The CIDR of Confluent Cloud Network
cidr = "10.10.0.0/16"

# The AWS Account ID of the peer VPC owner.
# You can find your AWS Account ID here (https://console.aws.amazon.com/billing/home?#/account) under My Account section of the AWS Management Console. Must be a 12 character string.
aws_account_id = "012345678901"

# The AWS VPC ID of the peer VPC that you're peering with Confluent Cloud.
# You can find your AWS VPC ID here (https://console.aws.amazon.com/vpc/) under Your VPCs section of the AWS Management Console. Must start with `vpc-`.
vpc_id = "vpc-abcdef0123456789a"

# The AWS VPC CIDR blocks or subsets.
# This must be from the supported CIDR blocks and must not overlap with your Confluent Cloud CIDR block or any other network peering connection VPC CIDR (learn more about the requirements [here](https://docs.confluent.io/cloud/current/networking/peering/aws-peering.html#vpc-peering-on-aws)).
# You can find AWS VPC CIDR [here](https://console.aws.amazon.com/vpc/) under **Your VPCs -> Target VPC -> Details** section of the AWS Management Console.
routes = ["172.31.0.0/16"]

# Add credentials and other settings to $HOME/.aws/config
# for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files

# Requirements of VPC Peering on AWS
# https://docs.confluent.io/cloud/current/networking/peering/aws-peering.html#vpc-peering-on-aws
