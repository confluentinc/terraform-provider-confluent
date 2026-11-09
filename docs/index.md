---
page_title: "Provider: Confluent"
subcategory: ""
description: |-
  
---

# Confluent Provider

Simplify Apache Kafka Terraform deployment with the Confluent Terraform Provider. Manage Environments, Kafka Clusters, Kafka Topics, Kafka ACLs, Service Accounts, and more in Confluent.

Use the Confluent provider to deploy and manage [Confluent Cloud](https://www.confluent.io/confluent-cloud/) infrastructure. You must provide appropriate credentials to use the provider. The navigation menu provides details about the resources that you can interact with (_Resources_), and a guide (_Guides_) for how you can get started.

[![asciicast](https://asciinema.org/a/580630.svg)](https://asciinema.org/a/580630)

## Example Usage

Terraform `0.13` and later:

```terraform
# Configure the Confluent Provider
terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "2.94.0"
    }
  }
}

# Option #1: Manage multiple clusters in the same Terraform workspace
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

# Option #2: Manage a single Kafka cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster for more details
provider "confluent" {
  kafka_id            = var.kafka_id                   # optionally use KAFKA_ID env var
  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var
  kafka_api_key       = var.kafka_api_key              # optionally use KAFKA_API_KEY env var
  kafka_api_secret    = var.kafka_api_secret           # optionally use KAFKA_API_SECRET env var
}
# Manage topics, ACLs, etc.

# Option #2: Manage a single Schema Registry cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster for more details
provider "confluent" {
  schema_registry_id            = var.schema_registry_id            # optionally use SCHEMA_REGISTRY_ID env var
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
  schema_registry_api_key       = var.schema_registry_api_key       # optionally use SCHEMA_REGISTRY_API_KEY env var
  schema_registry_api_secret    = var.schema_registry_api_secret    # optionally use SCHEMA_REGISTRY_API_SECRET env var
}
# Manage schemas, subjects, etc.
```

## Enable Confluent Cloud Access

Confluent Cloud requires API keys to manage access and authentication to different parts of the service. An API key consists of a key and a secret. You can create and manage API keys by using either the [Confluent Cloud CLI](https://docs.confluent.io/ccloud-cli/current/index.html) or the [Confluent Cloud Console](https://confluent.cloud/). Learn more about Confluent Cloud API Key access [here](https://docs.confluent.io/cloud/current/client-apps/api-keys.html#ccloud-api-keys).

## Provider Authentication

Confluent Terraform provider allows authentication by using environment variables or static credentials.

### Environment Variables

Run the following commands to set the `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment variables:

```shell
# Option #1: Manage multiple clusters in the same Terraform workspace
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"

# Option #2: Manage a single Kafka cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster for more details
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ export KAFKA_ID="<kafka_id>"
$ export KAFKA_REST_ENDPOINT="<kafka_rest_endpoint>"
$ export KAFKA_API_KEY="<kafka_api_key>"
$ export KAFKA_API_SECRET="<kafka_api_secret>"

# Option #2: Manage a single Schema Registry cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster for more details
$ export SCHEMA_REGISTRY_ID="<schema_registry_id>"
$ export SCHEMA_REGISTRY_REST_ENDPOINT="<schema_registry_rest_endpoint>"
$ export SCHEMA_REGISTRY_API_KEY="<schema_registry_api_key>"
$ export SCHEMA_REGISTRY_API_SECRET="<schema_registry_api_secret>"
```

-> **Note:** Quotation marks are required around the API key and secret strings.

### Static Credentials

You can also provide static credentials in-line directly, or by input variable (do not forget to declare the variables as [sensitive](https://learn.hashicorp.com/tutorials/terraform/sensitive-variables#refactor-database-credentials)):

```terraform
# Option #1: Manage multiple clusters in the same Terraform workspace
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

# Option #2: Manage a single Kafka cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster for more details
provider "confluent" {
  kafka_id            = var.kafka_id                   # optionally use KAFKA_ID env var
  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var
  kafka_api_key       = var.kafka_api_key              # optionally use KAFKA_API_KEY env var
  kafka_api_secret    = var.kafka_api_secret           # optionally use KAFKA_API_SECRET env var
}
# Manage topics, ACLs, etc.

# Option #2: Manage a single Schema Registry cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster for more details
provider "confluent" {
  schema_registry_id            = var.schema_registry_id            # optionally use SCHEMA_REGISTRY_ID env var
  schema_registry_rest_endpoint = var.schema_registry_rest_endpoint # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
  schema_registry_api_key       = var.schema_registry_api_key       # optionally use SCHEMA_REGISTRY_API_KEY env var
  schema_registry_api_secret    = var.schema_registry_api_secret    # optionally use SCHEMA_REGISTRY_API_SECRET env var
}
# Manage schemas, subjects, etc.
```

!> **Warning:** Hardcoding credentials into a Terraform configuration is not recommended. Hardcoded credentials increase the risk of accidentally publishing secrets to public repositories.

### OAuth Credentials

-> **Note:** Authentication using the `oauth` credentials block is available in **General Availability** for all customers. However, it's recommended to try OAuth features in non-production environments before rolling out to production.

Confluent Terraform provider allows authentication by using OAuth credentials. You can use the `oauth` block to configure the provider with OAuth credentials.

```shell
# Option #1: Provide Identity Provider client id and secret, token retrieval URL, and the established Identity Pool ID
provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url            # the URL to retrieve the token from your Identity Provider, such as "https://mycompany.okta.com/oauth2/abc123/v1/token"
    oauth_external_client_id  = var.oauth_external_client_id           # the client id of your Identity Provider authorization server
    oauth_external_client_secret = var.oauth_external_client_secret    # the client secret of your Identity Provider authorization server
    oauth_identity_pool_id = var.oauth_identity_pool_id                # the established Identity Pool ID on Confluent Cloud based on your Identity Provider
    oauth_external_token_scope = var.oauth_external_token_scope        # the application client scope, might be required by your Identity Provider, such as Microsoft Azure Entra ID requires "api://<client_id>/.default" scope
  }
}
# Token refresh capability is supported by Confluent Provider for Option #1.

# Option #2: Provide a static token from the Identity Provider the established Identity Pool ID
provider "confluent" {
  oauth {
    oauth_external_access_token = var.oauth_external_access_token    # the static access token from your Identity Provider, please ensure it is not expired
    oauth_identity_pool_id = var.oauth_identity_pool_id              # the established Identity Pool ID on Confluent Cloud based on your Identity Provider
  }
}
# Token refresh capability is NOT supported by Confluent Provider for Option #2.
```
Complete examples (with Okta and Microsoft Azure Entra ID as identity provider) for using OAuth credentials with the Confluent Terraform Provider can be found [here](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth).

-> **Note:** You still need `cloud_api_key` and `cloud_api_secret` to manage below Confluent Cloud resources/data-sources as they are not supported with OAuth credentials yet:
* `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_catalog_integration).
* `confluent_custom_connector_plugin` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_custom_connector_plugin).
* `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact).
* `confluent_tableflow_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tableflow_topic) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tableflow_topic).

-> **Note:** An Identity Provider must be set up first on Confluent Cloud before using the OAuth credentials for Terraform Provider. You can find more information about Identity Provider setting up [here](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.htmll).

-> **Note:** After Identity Provider is set up, an Identity Pool must be added and assigned proper RBAC roles to manage Confluent Cloud resources/data-sources with corresponding scope, more details can be found [here](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-pools.html).

-> **Note:** `oauth_external_token_scope` could be optional or required based on your Identity Provider. For example, Microsoft Azure Entra ID requires `api://<client_id>/.default` scope to retrieve the token, while Okta does not require any scope.

-> **Note:** To switch your Terraform configuration from API key/secret authentication to OAuth, update your provider block by removing any references to variables such as `cloud_api_key`, `flink_api_key`, `kafka_api_key`, `schema_registry_api_key`, and similar variables. Also, remove any `credentials` blocks from resources like `confluent_kafka_topic`, `confluent_schema`, and `confluent_flink_statement`. Instead, specify your authentication details within the `oauth {}` block. After making these changes, apply your configuration to start using OAuth.

!> **Warning:** Without proper Identity Provider setup, Identity Pool creation and RBAC roles assignment, the OAuth credentials will not work with Confluent Terraform Provider.

## Retries and Rate Limits

Confluent Cloud APIs are rate-limited. When a request is rate-limited (`429 Too Many Requests`) or fails with a server or network error, the provider retries it before failing the operation. Set `max_retries` (or the `TF_PROVIDER_CONFLUENT_MAX_RETRIES` environment variable) to change the number of retries:

```terraform
provider "confluent" {
  max_retries = 20 # optionally use TF_PROVIDER_CONFLUENT_MAX_RETRIES env var
}
```

Requests to the Connect, IAM, API keys and RBAC APIs, where large configurations are most often rate-limited, always retry at least 12 times. These requests are made by the `confluent_connector`, `confluent_service_account`, `confluent_api_key`, `confluent_role_binding`, `confluent_invitation` and `confluent_tf_importer` resources, and by the `confluent_service_account`, `confluent_user`, `confluent_users`, `confluent_role_binding` and `confluent_invitation` data sources.

| `max_retries`         | Connect, IAM, API keys and RBAC requests | All other requests    |
|-----------------------|------------------------------------------|-----------------------|
| Not set (default `4`) | 12 retries                               | 4 retries             |
| `4` to `12`           | 12 retries                               | `max_retries` retries |
| Above `12`            | `max_retries` retries                    | `max_retries` retries |

-> **Note:** `max_retries` must be at least `4`.

-> **Note:** If a large `terraform plan` or `terraform apply` still fails with `429 Too Many Requests`, lower Terraform's `-parallelism` (the default is 10) or raise `max_retries`, for example to `20`. Before each retry, a rate-limited request waits at least as long as the API asks in its `Retry-After` response header, which is 1 second for the per-second limits of the control-plane APIs at `api.confluent.cloud`. Other failures, and rate-limited requests without that header, wait 1, 2, 4, 8, 16 and then 30 seconds, so a higher `max_retries` also makes the provider take longer to report an API that is unavailable.

## Timeouts

Every create, read, update and delete operation has a time limit. The limit covers the whole operation: every API request it makes, the retries of those requests, and any wait for Confluent Cloud to finish the change. When the limit is reached, the operation fails, usually with a `context deadline exceeded` error. A few steps don't stop at the limit: the fixed waits noted on some resource pages, and file uploads, finish first, so an operation can run past a shorter limit before it fails.

Unless a resource's documentation says otherwise, each operation is limited to 20 minutes, and so is each data source read. Resources whose operations wait for Confluent Cloud to finish work in the background, such as provisioning a Dedicated Kafka cluster or deleting a network, set their own defaults for those operations and accept a `timeouts` block for them. The **Timeouts** section of the resource's page lists the operations you can set and their defaults. A value in the `timeouts` block can be higher or lower than the default:

```terraform
resource "confluent_kafka_cluster" "dedicated" {
  # ...

  timeouts {
    create = "96h"
  }
}
```

-> **Note:** A `timeouts` block accepts only the operations listed for that resource. Values use Go duration syntax, such as `90m` or `6h`.

-> **Note:** The limits apply to each operation, not to a whole `terraform apply`, which has no time limit. Importing a resource also has no time limit; the read that Terraform runs after an import is limited to 20 minutes.

-> **Note:** Terraform saves the `timeouts` values in state when it creates or updates a resource, and `terraform destroy` uses the saved values. An imported resource starts with the resource's default values. A change to only the `timeouts` block doesn't update the resource, so a new `delete` value takes effect after the resource's next create or update. Until then, a resource that was last created, updated or imported with a provider version that didn't support a `delete` timeout for it is deleted with the 20-minute limit.

-> **Note:** If a create runs out of time after Confluent Cloud has started creating the resource, Terraform usually saves the resource as tainted, and the next `terraform apply` destroys and re-creates it. If the resource finishes provisioning later, run `terraform untaint` to keep it. `confluent_api_key` and `confluent_schema_exporter` are exceptions: they're saved to state only after they're ready, so a create that runs out of time while waiting saves nothing. Run `terraform state list` to check. If the resource isn't there, delete the new API key, or import the schema exporter, before you apply again.

-> **Note:** If you interrupt `terraform apply` once, for example with Ctrl-C, operations already in progress keep waiting until they finish or reach their limit. Interrupting again stops Terraform right away, which can leave resources that were being created out of state.

## Helpful Links/Information

* [Report Bugs](https://github.com/confluentinc/terraform-provider-confluent/issues)

* [Request Features](mailto:cflt-tf-access@confluent.io?subject=Feature%20Request)

!> **Warning:** Terraform version `1.6.0` is not supported. See [this issue](https://github.com/confluentinc/terraform-provider-confluent/issues/315) for more details.

-> **Note:** If you are running into issues when trying to write a reusable module using this provider, please look at [this message](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/20#issuecomment-1011833161) to resolve the problem.

-> **Note:** It is recommended to set `lifecycle { prevent_destroy = true }` on production instances to prevent accidental instance deletion. This setting rejects plans that would destroy or recreate the instance, such as attempting to change uneditable attributes. Read more about it in the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).
