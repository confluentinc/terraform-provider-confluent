# The AWS account ID to enable for the Private Link Access.
# You can find your AWS account ID here (https://console.aws.amazon.com/billing/home?#/account) under My Account section of the AWS Management Console. Must be a 12 character string.
aws_account_id = "012345678901"

# The VPC ID that you want to connect to Confluent Cloud Cluster
# https://us-east-1.console.aws.amazon.com/vpc/home?region=us-east-1#vpcs:
vpc_id = "vpc-abcdef0123456789a"

# The region of your VPC that you want to connect to Confluent Cloud Cluster
# Cross-region AWS PrivateLink connections are not supported yet.
region = "us-east-1"

# The map of Zone ID to Subnet ID. You can find subnets to private link mapping information by clicking at VPC -> Subnets from your AWS Management Console (https://console.aws.amazon.com/vpc/home)
# https://us-west-1.console.aws.amazon.com/vpc/home?region=us-east-1#subnets:search=vpc-abcdef0123456789a
# You must have subnets in your VPC for these zones so that IP addresses can be allocated from them.
subnets_to_privatelink = {
  "use1-az1" = "subnet-0123456789abcdef0",
  "use1-az4" = "subnet-0123456789abcdef1",
  "use1-az5" = "subnet-0123456789abcdef2",
}

# Limitations of AWS PrivateLink
# https://docs.confluent.io/cloud/current/networking/private-links/aws-privatelink.html#limitations
