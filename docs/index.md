---
page_title: "Provider: Confluent"
subcategory: ""
description: |-
  
---

# Confluent Provider

Simplify Apache Kafka Terraform deployment with the Confluent Terraform Provider. Manage Environments, Kafka Clusters, Kafka Topics, Kafka ACLs, Service Accounts, and more in Confluent.

Use the Confluent provider to deploy and manage [Confluent Cloud](https://www.confluent.io/confluent-cloud/) infrastructure. You must provide appropriate credentials to use the provider. The navigation menu provides details about the resources that you can interact with (_Resources_), and a guide (_Guides_) for how you can get started.

[![asciicast](https://asciinema.org/a/534729.svg)](https://asciinema.org/a/534729)

## Example Usage

Terraform `0.13` and later:

```terraform
# Configure the Confluent Provider
terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "1.13.0"
    }
  }
}

# Option #1 when managing multiple clusters in the same Terraform workspace
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

# Option #2 when managing a single cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-cluster for more details
provider "confluent" {
  cloud_api_key       = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret    = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var

  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var 
  kafka_api_key       = var.kafka_api_key              # optionally use KAFKA_API_KEY env var
  kafka_api_secret    = var.kafka_api_secret           # optionally use KAFKA_API_SECRET env var
}
# Create the resources
```

## Enable Confluent Cloud Access

Confluent Cloud requires API keys to manage access and authentication to different parts of the service. An API key consists of a key and a secret. You can create and manage API keys by using either the [Confluent Cloud CLI](https://docs.confluent.io/ccloud-cli/current/index.html) or the [Confluent Cloud Console](https://confluent.cloud/). Learn more about Confluent Cloud API Key access [here](https://docs.confluent.io/cloud/current/client-apps/api-keys.html#ccloud-api-keys).

## Provider Authentication

Confluent Terraform provider allows authentication by using environment variables or static credentials.

### Environment Variables

Run the following commands to set the `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment variables:

```shell
# Option #1 when managing multiple clusters in the same Terraform workspace
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"

# Option #2 when managing a single cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-cluster for more details
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ export KAFKA_REST_ENDPOINT="<kafka_rest_endpoint>"
$ export KAFKA_API_KEY="<kafka_api_key>"
$ export KAFKA_API_SECRET="<kafka_api_secret>"
```

-> **Note:** Quotation marks are required around the API key and secret strings.

### Static Credentials

You can also provide static credentials in-line directly, or by input variable (do not forget to declare the variables as [sensitive](https://learn.hashicorp.com/tutorials/terraform/sensitive-variables#refactor-database-credentials)):

```terraform
# Option #1 when managing multiple clusters in the same Terraform workspace
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

# Option #2 when managing a single cluster in the same Terraform workspace
# See https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-cluster for more details
provider "confluent" {
  cloud_api_key       = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret    = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var

  kafka_rest_endpoint = var.kafka_rest_endpoint        # optionally use KAFKA_REST_ENDPOINT env var 
  kafka_api_key       = var.kafka_api_key              # optionally use KAFKA_API_KEY env var
  kafka_api_secret    = var.kafka_api_secret           # optionally use KAFKA_API_SECRET env var
}
```

!> **Warning:** Hardcoding credentials into a Terraform configuration is not recommended. Hardcoded credentials increase the risk of accidentally publishing secrets to public repositories.

## Helpful Links/Information

* [Report Bugs](https://github.com/confluentinc/terraform-provider-confluent/issues)

* [Request Features](mailto:cflt-tf-access@confluent.io?subject=Feature%20Request)

-> **Note:** If you are running into issues when trying to write a reusable module using this provider, please look at [this message](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/20#issuecomment-1011833161) to resolve the problem.

-> **Note:** It is recommended to set `lifecycle { prevent_destroy = true }` on production instances to prevent accidental instance deletion. This setting rejects plans that would destroy or recreate the instance, such as attempting to change uneditable attributes. Read more about it in the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).
