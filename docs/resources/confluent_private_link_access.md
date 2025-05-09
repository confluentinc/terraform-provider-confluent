---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_private_link_access Resource - terraform-provider-confluent"
subcategory: ""
description: |-
  
---

# confluent_private_link_access Resource

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

`confluent_private_link_access` provides a Private Link Access resource that enables creating and deleting access to PrivateLink endpoints by AWS account, Azure subscription, or GCP project ID.

-> **Note:** It is recommended to set `lifecycle { prevent_destroy = true }` on production instances to prevent accidental Private Link Access deletion. This setting rejects plans that would destroy or recreate the Private Link Access, such as attempting to change uneditable attributes. Read more about it in the [Terraform docs](https://www.terraform.io/language/meta-arguments/lifecycle#prevent_destroy).

## Example Usage

### Example Private Link Access on AWS

```terraform
resource "confluent_environment" "development" {
  display_name = "Development"
}

resource "confluent_network" "aws-private-link" {
  display_name     = "AWS Private Link Network"
  cloud            = "AWS"
  region           = "us-east-1"
  connection_types = ["PRIVATELINK"]
  zones            = ["use1-az1", "use1-az2", "use1-az6"]
  environment {
    id = confluent_environment.development.id
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_private_link_access" "aws" {
  display_name = "AWS Private Link Access"
  aws {
    account = "012345678901"
  }
  environment {
    id = confluent_environment.development.id
  }
  network {
    id = confluent_network.aws-private-link.id
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### Example Private Link Access on Azure

```terraform
resource "confluent_environment" "development" {
  display_name = "Development"
}

resource "confluent_network" "azure-private-link" {
  display_name     = "Azure Private Link Network"
  cloud            = "AZURE"
  region           = "centralus"
  connection_types = ["PRIVATELINK"]
  environment {
    id = confluent_environment.development.id
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_private_link_access" "azure" {
  display_name = "Azure Private Link Access"
  azure {
    subscription = "1234abcd-12ab-34cd-1234-123456abcdef"
  }
  environment {
    id = confluent_environment.development.id
  }
  network {
    id = confluent_network.azure-private-link.id
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

### Example Private Service Connect on GCP

```terraform
resource "confluent_environment" "development" {
  display_name = "Development"
}

resource "confluent_network" "gcp-private-service-connect" {
  display_name     = "GCP Private Service Connect Network"
  cloud            = "GCP"
  region           = "us-central1"
  connection_types = ["PRIVATELINK"]
  zones            = ["us-central1-a","us-central1-b","us-central1-c"]
  environment {
    id = confluent_environment.development.id
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "confluent_private_link_access" "gcp" {
  display_name = "GCP Private Service Connect"
  gcp {
    project = "temp-gear-123456"
  }
  environment {
    id = confluent_environment.development.id
  }
  network {
    id = confluent_network.gcp-private-service-connect.id
  }

  lifecycle {
    prevent_destroy = true
  }
}
```

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `display_name` - (Optional String) The name of the Private Link Access.
- `environment` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the Environment that the Private Link Access belongs to, for example, `env-abc123`.
- `network` (Required Configuration Block) supports the following:
    - `id` - (Required String) The ID of the Network that the Private Link Access belongs to, for example, `n-abc123`.
- `aws` - (Optional Configuration Block) The AWS-specific Private Link Access details if available. It supports the following:
    - `account` - (Required String) The AWS account ID to enable for the Private Link Access. You can find your AWS account ID [here] (https://console.aws.amazon.com/billing/home?#/account) under **My Account** in your AWS Management Console. Must be a **12 character string**.
- `azure` - (Optional Configuration Block) The Azure-specific Private Link Access details if available. It supports the following:
    - `subscription` - (Required String) The Azure subscription ID to enable for the Private Link Access. You can find your Azure subscription ID in the subscription section of your [Microsoft Azure Portal] (https://portal.azure.com/#blade/Microsoft_Azure_Billing/SubscriptionsBlade). Must be a valid **32 character UUID string**.
- `gcp` - (Optional Configuration Block) The GCP-specific Private Service Connect details if available. It supports the following:
  - `project` - (Required String) The GCP project ID to allow for Private Service Connect access. You can find your Google Cloud Project ID under **Project ID** section of your [Google Cloud Console dashboard](https://console.cloud.google.com/home/dashboard).

-> **Note:** Exactly one from the `aws`, `azure`, `gcp` configuration blocks must be specified.

-> **Note:** Learn more about Private Link Access limitations on AWS [here](https://docs.confluent.io/cloud/current/networking/private-links/aws-privatelink.html#limitations).

-> **Note:** Learn more about Private Link Access limitations on Azure [here](https://docs.confluent.io/cloud/current/networking/private-links/azure-privatelink.html#limitations).

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `id` - (Required String) The ID of the Private Link Access, for example, `pla-abc123`.

## Import

-> **Note:** `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment variables must be set before importing a Private Link Access.

You can import a Private Link Access by using Environment ID and Private Link Access ID, in the format `<Environment ID>/<Private Link Access ID>`. The following example shows how to import a Private Link Access:

```shell
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ terraform import confluent_private_link_access.my_pla env-abc123/pla-abc123
```

!> **Warning:** Do not forget to delete terminal command history afterwards for security purposes.

## Getting Started
The following end-to-end examples might help to get started with `confluent_private_link_access` resource:
  * [dedicated-privatelink-aws-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-acls): _Dedicated_ Kafka cluster on AWS that is accessible via PrivateLink connections with authorization using ACLs
  * [dedicated-privatelink-aws-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-rbac): _Dedicated_ Kafka cluster on AWS that is accessible via PrivateLink connections with authorization using RBAC
  * [dedicated-privatelink-azure-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-rbac): _Dedicated_ Kafka cluster on Azure that is accessible via PrivateLink connections with authorization using RBAC
  * [dedicated-privatelink-azure-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-acls): _Dedicated_ Kafka cluster on Azure that is accessible via PrivateLink connections with authorization using ACLs
  * [dedicated-private-service-connect-gcp-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-acls): _Dedicated_ Kafka cluster on GCP that is accessible via Private Service Connect connections with authorization using ACLs
  * [dedicated-private-service-connect-gcp-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-rbac): _Dedicated_ Kafka cluster on GCP that is accessible via Private Service Connect connections with authorization using RBAC
