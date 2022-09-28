# The GCP Project ID
customer_project_id = "temp-gear-123456"

# The region of Confluent Cloud Network
region = "us-central1"

# A map of Zone to Subnet Name
subnet_name_by_zone = {
  "us-central1-a" = "default",
  "us-central1-b" = "default",
  "us-central1-c" = "default",
}

# The VPC network name that you want to connect to Confluent Cloud Cluster
customer_vpc_network = "default"

# The subnetwork name that you want to connect to Confluent Cloud Cluster
customer_subnetwork_name = "default"

# Set GOOGLE_APPLICATION_CREDENTIALS environment variable to a path to a key file
# for Google TF Provider to work: https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/getting_started#adding-credentials

# Limitations of Private Service Connect
# https://docs.confluent.io/cloud/current/networking/private-links/gcp-private-service-connect.html#limitations
