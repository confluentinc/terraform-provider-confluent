# Azure Provider Integration (v2) Example

This example demonstrates how to create and configure an Azure provider integration using the new v2 API with the two-resource approach.

## Overview

The v2 provider integration uses two resources for a seamless one-step apply experience:

1. **`confluent_provider_integration_v2`**: Creates the integration in DRAFT status
2. **`confluent_provider_integration_v2_authorization`**: Validates the Azure configuration

## Prerequisites

Before running this example:

1. **Azure Setup**:
   - Have an Azure Active Directory tenant
   - Note your Azure Tenant ID
   - Ensure you have permissions to grant access to Confluent's multi-tenant application

2. **Confluent Cloud Setup**:
   - Confluent Cloud API Key and Secret
   - Environment ID where you want to create the integration

## Usage

1. **Set variables**:
   ```bash
   export TF_VAR_confluent_cloud_api_key="<your-api-key>"
   export TF_VAR_confluent_cloud_api_secret="<your-api-secret>"
   export TF_VAR_environment_id="env-xxxxx"
   export TF_VAR_azure_tenant_id="12345678-1234-1234-1234-123456789abc"
   ```

2. **Apply Terraform** (creates integration and shows setup instructions):
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. **Complete Azure setup** (follow the warning instructions):
   - Note the `confluent_multi_tenant_app_id` from the output
   - Grant admin consent: Visit the admin consent URL from the warning
   - Create service principal: `az ad sp create --id <app-id>`
   - In Azure Portal → Enterprise Applications → grant permissions

4. **Validate the integration**:
   ```bash
   terraform apply  # Warning should disappear once Azure setup is complete
   ```

## What This Creates

- A provider integration in DRAFT status
- Validation of the Azure configuration
- Output of the Confluent Multi-Tenant App ID for Azure setup

## Next Steps

After running this example:

1. Note the `confluent_multi_tenant_app_id` from the output
2. In Azure, grant this application appropriate permissions to your Azure resources
3. Use this provider integration with your connectors

## Two-Resource Approach

This example uses the MongoDB-inspired two-resource pattern to enable one-step apply:

- **Setup Resource**: Creates the integration framework
- **Authorization Resource**: Handles configuration and validation

This approach eliminates the need for manual Terraform configuration changes between apply operations.

## Notes

- The v2 API currently supports validation only
- PATCH operations will be added when the backend API supports them
- This example focuses on Azure; GCP support follows the same pattern