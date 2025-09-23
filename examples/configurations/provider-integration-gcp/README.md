### Notes

1. Set the required variables in `terraform.tfvars`:
   ```
   confluent_cloud_api_key    = "<your-api-key>"
   confluent_cloud_api_secret = "<your-api-secret>"
   environment_id             = "env-xxxxx"
   gcp_service_account       = "my-sa@my-project.iam.gserviceaccount.com"
   ```

2. After `terraform apply`, follow the warning instructions to complete GCP setup:
   - Run the `gcloud projects add-iam-policy-binding` command from the warning
   - Grant your service account necessary permissions for your connectors
   - Re-run `terraform apply` to validate

3. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.

4. See [Provider Integration Documentation](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_provider_integration_setup) for more details.