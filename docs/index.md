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
      version = "2.26.0"
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

-> **Note:** Authentication using the `oauth` credentials block is available in **Early Access** for early adopters. Early Access features are introduced to gather customer feedback. This feature should be used only for evaluation and non-production testing purposes or to provide feedback to Confluent, particularly as it becomes more widely available in follow-on editions.
**Early Access** features are intended for evaluation use in development and testing environments only, and not for production use. The warranty, SLA, and Support Services provisions of your agreement with Confluent do not apply to Early Access features. Early Access features are considered to be a Proof of Concept as defined in the Confluent Cloud Terms of Service. Confluent may discontinue providing preview releases of the Early Access features at any time in Confluent’s sole discretion.

Confluent Terraform provider allows authentication by using OAuth credentials. You can use the `oauth` block to configure the provider with OAuth credentials.

```shell
# Option #1: Provide Identity Provider client id and secret, token retrieval URL, and the established Identity Pool ID
provider "confluent" {
  oauth {
    oauth_external_token_url = var.oauth_external_token_url            # the URL to retrieve the token from your Identity Provider, such as "https://mycompany.okta.com/oauth2/abc123/v1/token"
    oauth_external_client_id  = var.oauth_external_client_id           # the client id of your Identity Provider authorization server
    oauth_external_client_secret = var.oauth_external_client_secret    # the client secret of your Identity Provider authorization server
    oauth_identity_pool_id = var.oauth_identity_pool_id                # the established Identity Pool ID on Confluent Cloud based on your Identity Provider
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
A complete example for using OAuth credentials with the Confluent Terraform Provider can be found [here](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth).

-> **Note:** You still need `cloud_api_key` and `cloud_api_secret` to manage below Confluent Cloud resources/data-sources as they are not supported with OAuth credentials yet:
* `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key).
* `confluent_catalog_integration` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_integration) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_catalog_integration).
* `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_cluster_link).
* `confluent_custom_connector_plugin` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_custom_connector_plugin).
* `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact).
* `confluent_tableflow_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tableflow_topic) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tableflow_topic).

-> **Note:** An Identity Provider must be set up first on Confluent Cloud before using the OAuth credentials for Terraform Provider. You can find more information about Identity Provider setting up [here](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-providers.htmll).

-> **Note:** After Identity Provider is set up, an Identity Pool must be added and assigned proper RBAC roles to manage Confluent Cloud resources/data-sources with corresponding scope, more details can be found [here](https://docs.confluent.io/cloud/current/security/authenticate/workload-identities/identity-providers/oauth/identity-pools.html).

!> **Warning:** Without proper Identity Provider setup, Identity Pool creation and RBAC roles assignment, the OAuth credentials will not work with Confluent Terraform Provider.

## Helpful Links/Information

* [Report Bugs](https://github.com/confluentinc/terraform-provider-confluent/issues)

* [Request Features](mailto:cflt-tf-access@confluent.io?subject=Feature%20Request)

!> **Warning:** Terraform version `1.6.0` is not supported. See [this issue](https://github.com/confluentinc/terraform-provider-confluent/issues/315) for more details.

-> **Note:** If you are running into issues when trying to write a reusable module using this provider, please look at [this message](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/20#issuecomment-1011833161) to resolve the problem.

-> **Note:** It is recommended to set `lifecycle { prevent_destroy = true }` on production instances to prevent accidental instance deletion. This setting rejects plans that would destroy or recreate the instance, such as attempting to change uneditable attributes. Read more about it in the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).
