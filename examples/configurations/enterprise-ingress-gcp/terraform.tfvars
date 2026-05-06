confluent_cloud_api_key    = "<confluent-cloud-api-key>"
confluent_cloud_api_secret = "<confluent-cloud-api-secret>"

region           = "us-central1"
resource_prefix  = "test"
environment_name = "gcp-ingress-test"

# Create this in GCP first (via Console or gcloud), connecting to the service_attachment from the gateway output.
private_service_connect_connection_id = "116002050319319045"
