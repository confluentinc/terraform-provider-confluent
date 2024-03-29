---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_transit_gateway_attachment Data Source - terraform-provider-confluent"
subcategory: ""
description: |-
  
---

# confluent_transit_gateway_attachment Data Source

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

`confluent_transit_gateway_attachment` describes a Transit Gateway Attachment data source.

## Example Usage

```terraform
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

data "confluent_transit_gateway_attachment" "example_using_id" {
  id = "tgwa-abc123"
  environment {
    id = "env-xyz456"
  }
}
output "example_using_id" {
  value = data.confluent_transit_gateway_attachment.example_using_id
}
data "confluent_transit_gateway_attachment" "example_using_name" {
  display_name = "my_tgwa"
  environment {
    id = "env-xyz456"
  }
}
output "example_using_name" {
  value = data.confluent_transit_gateway_attachment.example_using_name
}
```

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `id` - (Optional String) The ID of the Peering, for example, `tgwa-abc123`.
- `display_name` - (Optional String) A human-readable name for the Transit Gateway Attachment.
- `environment` (Required Configuration Block) supports the following:
  - `id` - (Required String) The ID of the Environment that the Transit Gateway Attachment belongs to, for example, `env-xyz456`.

-> **Note:** Exactly one from the `id` and `display_name` attributes must be specified.

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `id` - (Required String) The ID of the Transit Gateway Attachment, for example, `tgwa-abc123`.
- `display_name` - (Required String) The name of the Transit Gateway Attachment.
- `environment` (Required Configuration Block) supports the following:
  - `id` - (Required String) The ID of the Environment that the Transit Gateway Attachment belongs to, for example, `env-abc123`.
- `network` (Required Configuration Block) supports the following:
  - `id` - (Required String) The ID of the Network that the Transit Gateway Attachment belongs to, for example, `n-abc123`.
- `aws` - (Required Configuration Block) The AWS-specific Transit Gateway Attachment details. It supports the following:
  - `ram_resource_share_arn` - (Required String) The Amazon Resource Name (ARN) of the Resource Access Manager (RAM) Resource Share of the transit gateway your Confluent Cloud network attaches to.
  - `transit_gateway_id` - (Required String) The ID of the AWS Transit Gateway that you want Confluent CLoud to be attached to. Must start with `tgw-`.
  - `routes` - (Required List of String) List of destination routes for traffic from Confluent VPC to customer VPC via Transit Gateway.
  - `transit_gateway_attachment_id` - (Required String) The ID of the AWS Transit Gateway VPC Attachment that attaches Confluent VPC to Transit Gateway.

-> **Note:** Use the `aws[0]` prefix for referencing these attributes, for example, `data.confluent_transit_gateway_attachment.example_using_name.aws[0].routes`.
