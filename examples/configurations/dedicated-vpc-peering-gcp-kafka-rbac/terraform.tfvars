# Cross-region access to Confluent Cloud is not supported when VPC peering is enabled with Google Cloud. Your VPC subnets and Confluent Cloud must be in the same region.
# The region of Confluent Cloud Network
region = ""

# The region of the GCP peer VPC network
customer_region = ""

# The CIDR of Confluent Cloud Network
cidr = "10.10.0.0/16"

# The GCP Project ID
customer_project_id = ""

# The VPC network name that you're peering to Confluent Cloud
customer_vpc_network = ""

# The name of the peering on GCP that will be created via TF
customer_peering_name = ""

# Set GOOGLE_APPLICATION_CREDENTIALS environment variable to a path to a key file
# for Google TF Provider to work: https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#adding-credentials

# Requirements of VPC Peering on GCP
# https://docs.confluent.io/cloud/current/networking/peering/gcp-peering.html#vpc-peering-on-gcp
