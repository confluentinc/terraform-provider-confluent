### Notes

1. Set the required variables in `terraform.tfvars`:
   ```
   confluent_cloud_api_key    = "<your-api-key>"
   confluent_cloud_api_secret = "<your-api-secret>"
   environment_id             = "env-xxxxx"
   azure_tenant_id           = "12345678-1234-1234-1234-123456789abc"
   ```

2. After `terraform apply`, follow the warning instructions to complete Azure setup:
   - Run the `az ad sp create` command from outputs
   - Grant permissions in Azure Portal → Enterprise Applications
   - Re-run `terraform apply` to validate

3. See [Sample Project for Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that provides step-by-step instructions of running this example.

4. See [Provider Integration Documentation](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_provider_integration_v2) for more details.