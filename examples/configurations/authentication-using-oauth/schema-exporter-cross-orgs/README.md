# Confluent Terraform Example – Cross-Organization Schema Exporter

This example demonstrates how to use the [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs) to configure a schema exporter [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter) across **two organizations** (`Source` and `Destination`),
exporting schemas from a source Schema Registry cluster to a destination Schema Registry cluster in a different Confluent Cloud organization.

---

## Features

- **Provider Configuration**  
  Uses OAuth-based authentication with external token exchange.

- **Cross-Organization Export**
  Exports schemas from a source Schema Registry cluster in one organization to a destination Schema Registry cluster in another organization.

- **Schema Exporter**
  Configures a schema exporter to replicate schemas from the source Schema Registry to the destination Schema Registry using OAuth authentication.

---

## Prerequisites

1. [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 0.13
2. [Confluent Cloud account](https://confluent.cloud/) with appropriate API/Service account credentials
3. An external OAuth provider (e.g., Azure AD, Okta, etc.) configured, see [instructions](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.html) for details
4. A valid **OAuth identity pool** configured in Confluent Cloud with appropriate roles assigned, see [instructions](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-pools.html) for details

---

## Variables

This example requires the following variables to be set, via `terraform.tfvars`:

- `oauth_external_token_url` – External token URL for fetching an OAuth token
- `oauth_external_client_id` – OAuth client ID
- `oauth_external_client_secret` – OAuth client secret
- `oauth_source_identity_pool_id` – Confluent Cloud identity pool ID for the source organization
- `oauth_destination_identity_pool_id` – Confluent Cloud identity pool ID for the destination organization
- `oauth_external_token_scope` - (Optional) the application client scope, might be required by your Identity Provider, such as Microsoft Azure Entra ID requires `api://<client_id>/.default` scope
- `source_schema_registry_cluster_id` – The ID of the source Schema Registry cluster (e.g., `lsrc-abc123`)
- `source_schema_registry_rest_endpoint` – The REST endpoint of the source Schema Registry cluster
- `destination_schema_registry_cluster_id` – The ID of the destination Schema Registry cluster (e.g., `lsrc-xyz789`)
- `destination_schema_registry_rest_endpoint` – The REST endpoint of the destination Schema Registry cluster
- Okta example can be found in [here](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth/okta)
- Microsoft Azure Entra ID example can be found in [here](https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/authentication-using-oauth/azure-entra-id/terraform.tfvars)
