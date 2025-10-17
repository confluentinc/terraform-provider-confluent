# Azure Blob Storage Sink Connector with Managed Identity

This example demonstrates how to create an Azure Blob Storage Sink connector using the **Provider Integration v2 API** with Managed Identity authentication.

## Overview

This configuration:
1. Creates a Confluent Cloud environment with Schema Registry
2. Creates a Kafka cluster on Azure
3. Sets up necessary service accounts and ACLs
4. Creates a Provider Integration using the **v2 API** (Azure/GCP support)
5. Creates Azure resources (Service Principal with Federated Identity)
6. Creates an Azure Blob Storage Sink connector using Managed Identity authentication

## Provider Integration v2 Flow

The v2 API uses a two-step process:

1. **Setup** (`confluent_provider_integration_setup`): Creates the integration in DRAFT status via POST API
2. **Authorization** (`confluent_provider_integration_authorization`): Configures customer Azure tenant ID via PATCH API and validates the integration

This approach allows you to:
- Get the Confluent Multi-Tenant App ID before creating Azure resources
- Use that App ID to create the necessary Azure Service Principal
- Validate the connection once Azure resources are set up

## Prerequisites

1. **Confluent Cloud Account**
   - Cloud API Key and Secret
   - Sufficient permissions to create environments, clusters, and connectors

2. **Azure Account**
   - Tenant ID and Subscription ID
   - Permissions to create:
     - Azure AD Applications and Service Principals
     - Federated Identity Credentials
     - Storage Account and Container (or use existing)
     - IAM role assignments

3. **Azure CLI** (optional, for manual verification):
   ```bash
   az login --tenant <your-tenant-id>
   ```

4. **Terraform**
   - Version 1.0+
   - Azure provider configured

## Configuration

### 1. Create `terraform.tfvars` file:

```hcl
confluent_cloud_api_key    = "your-confluent-api-key"
confluent_cloud_api_secret = "your-confluent-api-secret"

azure_tenant_id             = "12345678-1234-1234-1234-123456789abc"
azure_subscription_id       = "87654321-4321-4321-4321-cba987654321"
azure_region                = "eastus"
azure_storage_account_name  = "yourstorageaccount"
azure_resource_group_name   = "your-resource-group"
azure_container_name        = "kafka-sink-data"
```

### 2. Review and customize `main.tf`:
- Update Kafka cluster configuration (cloud, region, availability)
- Adjust connector configuration (topics, data format, flush size)
- Modify ACL settings based on your security requirements

## Usage

### Initialize Terraform:
```bash
terraform init
```

### Plan the deployment:
```bash
terraform plan
```

### Apply the configuration:
```bash
terraform apply
```

The first apply may show a warning about Azure setup being incomplete. This is expected - the resources will be created, but validation will complete once Azure resources are provisioned.

### View outputs:
```bash
terraform output
```

## How It Works

### Step 1: Provider Integration Setup
```hcl
resource "confluent_provider_integration_setup" "azure" {
  display_name = "azure_blob_connector_integration"
  cloud        = "AZURE"
  environment {
    id = confluent_environment.staging.id
  }
}
```
This creates the integration in **DRAFT** status and returns an integration ID.

### Step 2: Provider Integration Authorization
```hcl
resource "confluent_provider_integration_authorization" "azure" {
  provider_integration_id = confluent_provider_integration_setup.azure.id
  
  environment {
    id = confluent_environment.staging.id
  }
  
  azure {
    customer_azure_tenant_id = var.azure_tenant_id
  }
}
```
This:
- PATCHes the integration with your Azure Tenant ID
- Returns the `confluent_multi_tenant_app_id` 
- Validates the connection (may show warning until Azure resources are created)

### Step 3: Create Azure Resources
The `azure_resources_module` creates:
- Azure AD Application
- Service Principal
- Federated Identity Credential (OIDC trust for Confluent)
- IAM role assignments for Storage Blob Data Contributor

### Step 4: Create the Connector
The connector is created with `authentication.method = "Managed Identity"` and uses the provider integration ID.

## Verification

### 1. Check Provider Integration status:
```bash
confluent provider-integration list --environment <env-id>
confluent provider-integration describe <integration-id> --environment <env-id>
```

### 2. Check Azure resources:
```bash
# View Service Principal
az ad sp show --id <confluent-multi-tenant-app-id>

# View Federated Identity Credential
az ad app federated-credential list --id <application-object-id>

# Check Storage Account permissions
az role assignment list --scope /subscriptions/<sub-id>/resourceGroups/<rg>/providers/Microsoft.Storage/storageAccounts/<account-name>
```

### 3. Test the connector:
```bash
# Produce test data to the Kafka topic
confluent kafka topic produce orders --cluster <cluster-id>

# Check connector status
confluent connect cluster describe <connector-id> --cluster <kafka-cluster-id>

# Verify data in Azure Blob Storage
az storage blob list --account-name <storage-account> --container-name <container>
```

## Troubleshooting

### Validation warnings
If you see warnings about Azure setup being incomplete:
1. Check that the Service Principal was created: `az ad sp show --id <app-id>`
2. Verify Federated Identity Credential exists
3. Check IAM role assignments on the storage account
4. Wait 1-2 minutes for Azure IAM changes to propagate
5. Re-run `terraform apply` to revalidate

### Connector fails to start
- Check connector logs in Confluent Cloud UI
- Verify the storage account and container exist
- Confirm IAM permissions are correct
- Check that the provider integration is in CREATED/ACTIVE status

### Azure authentication errors
- Ensure the Azure Tenant ID is correct
- Verify Federated Identity Credential issuer is `https://token.confluent.cloud`
- Check the subject matches the expected format
- Confirm audience is `api://AzureADTokenExchange`

## Cleanup

```bash
terraform destroy
```

This will remove all Confluent Cloud resources and Azure resources created by the module.

## Notes

1. The Provider Integration v2 API currently supports **AZURE** and **GCP** only. For AWS, use the v1 `confluent_provider_integration` resource.
2. Azure IAM changes can take 1-7 minutes to propagate. If validation fails initially, wait and re-run `terraform apply`.
3. The `confluent_provider_integration_authorization` resource will show warnings (not errors) until Azure setup is complete, allowing you to get the necessary IDs to complete the setup.

## Learn More

- [Confluent Provider Integration Documentation](https://docs.confluent.io/cloud/current/connectors/provider-integration/index.html)
- [Azure Blob Storage Sink Connector](https://docs.confluent.io/cloud/current/connectors/cc-azure-blob-storage-sink.html)
- [Azure Federated Identity Credentials](https://learn.microsoft.com/en-us/azure/active-directory/develop/workload-identity-federation)

