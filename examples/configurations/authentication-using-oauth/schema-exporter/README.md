# Confluent Terraform Example – Multi-Environment Setup with Schema Exporter

This example demonstrates how to use the [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs) to provision Confluent Cloud resources across **two environments** (`Source` and `Destination`), 
including Kafka clusters, topics, schemas, and use a schema exporter [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter) to export schema from source cluster to destination cluster.

---

## Features

- **Provider Configuration**  
  Uses OAuth-based authentication with external token exchange.

- **Environments**  
  Creates two environments (`Source` and `Destination`) with **Stream Governance Essentials** enabled.

- **Kafka Clusters**  
  Provisions single-zone **Basic** Kafka clusters in separate AWS regions (`us-east-2` and `us-east-1`).

- **Schema Registry**  
  Fetches Schema Registry clusters for each environment.

- **Topics**  
  Creates source (`purchase_source`) and destination (`purchase_destination`) Kafka topics.

- **Schema Management**  
  Registers a local **Protobuf schema** (`purchase.proto`) bound to the source topic.

- **Schema Exporter**  
  Configures a schema exporter to replicate the registered schema from the source Schema Registry to the destination Schema Registry.

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
- `oauth_identity_pool_id` – Confluent Cloud identity pool ID
- `oauth_external_token_scope` - (Optional) the application client scope, might be required by your Identity Provider, such as Microsoft Azure Entra ID requires `api://<client_id>/.default` scope
- Okta example can be found in [here](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth/okta)
- Microsoft Azure Entra ID example can be found in [here](https://github.com/confluentinc/terraform-provider-confluent/blob/master/examples/configurations/authentication-using-oauth/azure-entra-id/terraform.tfvars)
