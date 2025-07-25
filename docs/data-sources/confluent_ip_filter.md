---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_ip_filter Data Source - terraform-provider-confluent"
subcategory: ""
description: |-
  
---

# confluent_ip_filter Data Source

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

`confluent_ip_filter` describes an IP Filter data source.

-> **Note:** See [IP Filtering on Confluent Cloud](https://docs.confluent.io/cloud/current/security/access-control/ip-filtering/overview.html) for more details about the IP Filtering feature, its prerequisites, and its limitations.

## Example Usage

```terraform
provider "confluent" {
  cloud_api_key    = var.confluent_cloud_api_key    # optionally use CONFLUENT_CLOUD_API_KEY env var
  cloud_api_secret = var.confluent_cloud_api_secret # optionally use CONFLUENT_CLOUD_API_SECRET env var
}

data "confluent_ip_filter" "example" {
  id = "ipf-abc123"
}

output "example" {
  value = data.confluent_ip_filter.example
}
```

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `id` - (Required String) The ID of the IP Group (e.g., `ipf-abc123`).

## Attributes Reference

The following attributes are exported:

- `filter_name` - (Required String) A human-readable name for an IP Filter. Can contain any unicode letter or number, the ASCII space character, or any of the following special characters: `[`, `]`, `|`, `&`, `+`, `-`, `_`, `/`, `.`, `,`.
- `resource_group` - (Required String) Scope of resources covered by this IP Filter. Available resource groups include `"management"` and `"multiple"`.
- `resource_scope` - (Required String) A CRN that specifies the scope of the IP Filter, specifically the organization or environment. Without specifying this property, the IP Filter would apply to the whole organization. For example, `"crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa"` or `data.confluent_organization.resource_name`.
- `operation_groups` - (Required List of Strings) Scope of resources covered by this IP Filter. Resource group must be set to 'multiple' in order to use this property. During update operations, note that the operation groups passed in will replace the list of existing operation groups (passing in an empty list will remove all operation groups) from the filter (in line with the behavior for `ip_groups` attribute).
- `ip_groups` - (Required List of Strings) A list of IP Groups.
