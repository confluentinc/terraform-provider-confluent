# GCP Provider Integration (v2) Example

This example demonstrates how to create and configure a GCP provider integration using the new v2 API with the two-resource approach.

## Overview

The v2 provider integration uses two resources for a seamless one-step apply experience:

1. **`confluent_provider_integration_v2`**: Creates the integration in DRAFT status
2. **`confluent_provider_integration_v2_authorization`**: Validates the GCP configuration

## Prerequisites

Before running this example:

1. **GCP Setup**:
   - Have a Google Cloud Project
   - Create a Service Account in your project
   - Note the Service Account email (e.g., `my-sa@my-project.iam.gserviceaccount.com`)
   - Ensure you have permissions to grant IAM roles

2. **Confluent Cloud Setup**:
   - Confluent Cloud API Key and Secret
   - Environment ID where you want to create the integration

## Usage

1. **Set variables**:
   ```bash
   export TF_VAR_confluent_cloud_api_key="<your-api-key>"
   export TF_VAR_confluent_cloud_api_secret="<your-api-secret>"
   export TF_VAR_environment_id="env-xxxxx"
   export TF_VAR_gcp_service_account="my-sa@my-project.iam.gserviceaccount.com"
   export TF_VAR_autovalidate="false"  # Start with false
   ```

2. **Initial apply** (gets multi-tenant app ID):
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

3. **Complete GCP IAM setup**:
   - Note the `confluent_service_account` from the output
   - In GCP Console → IAM & Admin → IAM
   - Grant the Confluent service account "Service Account Token Creator" role on your service account
   - Grant your service account the necessary permissions (e.g., BigQuery Data Editor)

4. **Validate the integration**:
   ```bash
   export TF_VAR_autovalidate="true"
   terraform apply
   ```

## What This Creates

- A provider integration in DRAFT status
- Validation of the GCP configuration
- Output of the Confluent Service Account for GCP IAM setup

## Next Steps

After running this example:

1. Note the `confluent_service_account` from the output
2. In GCP, grant this service account "Service Account Token Creator" role on your service account
3. Use this provider integration with your connectors

## Two-Resource Approach

This example uses the MongoDB-inspired two-resource pattern to enable one-step apply:

- **Setup Resource**: Creates the integration framework
- **Authorization Resource**: Handles configuration and validation

This approach eliminates the need for manual Terraform configuration changes between apply operations.

## Notes

- The v2 API currently supports validation only
- PATCH operations will be added when the backend API supports them
- This example focuses on GCP; Azure support follows the same pattern