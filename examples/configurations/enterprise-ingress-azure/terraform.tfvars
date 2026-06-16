confluent_cloud_api_key    = "<confluent-cloud-api-key>"
confluent_cloud_api_secret = "<confluent-cloud-api-secret>"

region           = "centralus"
resource_prefix  = "test"
environment_name = "azure-ingress-test"

# Create this in Azure first, pointing to the private_link_service_alias from the gateway output.
private_endpoint_resource_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg/providers/Microsoft.Network/privateEndpoints/my-pe"
