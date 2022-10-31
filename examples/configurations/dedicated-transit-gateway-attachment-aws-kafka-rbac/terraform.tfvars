# Cross-region access to Confluent Cloud is not supported when VPC peering is enabled with Google Cloud. Your AWS VPC and Confluent Cloud must be in the same region.
# The region of your VPC that you want to connect to Confluent Cloud Cluster
# Cross-region AWS PrivateLink connections are not supported yet.
region = "us-east-1"
# The region of the AWS peer VPC.
customer_region = "us-east-1"

# The CIDR of Confluent Cloud Network
cidr = "10.10.0.0/16"

# The AWS Account ID of the VPC owner.
# You can find your AWS Account ID here (https://console.aws.amazon.com/billing/home?#/account) under My Account section of the AWS Management Console. Must be a 12 character string.
aws_account_id = "012345678901"

# The AWS VPC ID of the VPC that you're connecting with Confluent Cloud.
# You can find your AWS VPC ID here (https://console.aws.amazon.com/vpc/) under Your VPCs section of the AWS Management Console. Must start with `vpc-`.
vpc_id = "vpc-abcdef0123456789a"

# The AWS Transit Gateway ID that is configured and ready to use with Confluent Cloud.
# To create a transit gateway, see, see https://docs.aws.amazon.com/vpc/latest/tgw/tgw-transit-gateways.html#create-tgw
# You can find your AWS Transit Gateway ID here (https://console.aws.amazon.com/vpc/) under Transit Gateways section of the AWS Management Console. Must start with `tgw-`.
transit_gateway_id = "tgw-abcdef0123456789a"

# Add credentials and other settings to $HOME/.aws/config
# for AWS TF Provider to work: https://registry.terraform.io/providers/hashicorp/aws/latest/docs#shared-configuration-and-credentials-files

# Limitations of AWS Transit Gateway
# https://docs.confluent.io/cloud/current/networking/aws-transit-gateway.html#limitations
