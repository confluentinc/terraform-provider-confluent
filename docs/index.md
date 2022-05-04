---
page_title: "Provider: Confluent"
subcategory: ""
description: |-
  
---

# Confluent Provider

Simplify Apache Kafka Terraform deployment with the Confluent Terraform Provider. Manage Environments, Kafka Clusters, Kafka Topics, Kafka ACLs, Service Accounts, and more in Confluent.

Use the Confluent provider to deploy and manage [Confluent Cloud](https://www.confluent.io/confluent-cloud/) infrastructure. You must provide appropriate credentials to use the provider. The navigation menu provides details about the resources that you can interact with (_Resources_), and a guide (_Guides_) for how you can get started.

-> **Note:** The Confluent Terraform provider is available in an **Preview Program** for early adopters. Preview features are introduced to gather customer feedback. This feature should be used only for evaluation and non-production testing purposes or to provide feedback to Confluent, particularly as it becomes more widely available in follow-on editions.  
**Preview Program** features are intended for evaluation use in development and testing environments only, and not for production use. The warranty, SLA, and Support Services provisions of your agreement with Confluent do not apply to Preview Program features. Preview Program features are considered to be a Proof of Concept as defined in the Confluent Cloud Terms of Service. Confluent may discontinue providing preview releases of the Preview Program features at any time in Confluent’s sole discretion.

[![asciicast](https://asciinema.org/a/491871.svg)](https://asciinema.org/a/491871)

## Example Usage

Terraform `0.13` and later:

```terraform
# Configure the Confluent Provider
terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "0.7.0"
    }
  }
}

provider "confluent" {
  api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
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
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
```

-> **Note:** Quotation marks are required around the API key and secret strings.

### Static Credentials

You can also provide static credentials in-line directly, or by input variable (do not forget to declare the variables as [sensitive](https://learn.hashicorp.com/tutorials/terraform/sensitive-variables#refactor-database-credentials)):

```terraform
provider "confluent" {
  api_key    = var.confluent_cloud_api_key
  api_secret = var.confluent_cloud_api_secret
}
```

!> **Warning:** Hardcoding credentials into a Terraform configuration is not recommended. Hardcoded credentials increase the risk of accidentally publishing secrets to public repositories.

## Helpful Links/Information

* [Report Bugs](https://github.com/confluentinc/terraform-provider-confluent/issues)

* [Request Features](mailto:cflt-tf-access@confluent.io?subject=Feature%20Request)

-> **Note:** If you can see `Error: 429 Too Many Requests` when running `terraform plan` or `terraform apply`, please follow [this piece of advice](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/15#issuecomment-972131964).

-> **Note:** If you are running into issues when trying to write a reusable module using this provider, please look at [this message](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/20#issuecomment-1011833161) to resolve the problem.
