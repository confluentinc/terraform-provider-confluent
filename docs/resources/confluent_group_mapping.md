---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "confluent_group_mapping Resource - terraform-provider-confluent"
subcategory: ""
description: |-
  
---

# confluent_group_mapping Resource

[![General Availability](https://img.shields.io/badge/Lifecycle%20Stage-General%20Availability-%2345c6e8)](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

`confluent_group_mapping` provides a Group Mapping resource that enables creating, editing, and deleting group mappings on Confluent Cloud.

-> **Note:** See [Group Mapping in Confluent Cloud](https://docs.confluent.io/cloud/current/access-management/authenticate/sso/group-mapping/overview.html) for more details.

## Example Usage

```terraform
resource "confluent_group_mapping" "application-developers" {
  display_name = "Application Developers"
  description  = "Admin access to production environment for Engineering"
  filter       = "\"engineering\" in groups"
}

resource "confluent_role_binding" "envadmin" {
  principal   = "User:${confluent_group_mapping.application-developers.id}"
  role_name   = "EnvironmentAdmin"
  crn_pattern = data.confluent_environment.prod.resource_name
}
```

<!-- schema generated by tfplugindocs -->
## Argument Reference

The following arguments are supported:

- `display_name` - (Required String) The name of the Group Mapping.
- `filter` - (Required String) A single group identifier or a condition based on [supported CEL operators](https://docs.confluent.io/cloud/current/access-management/authenticate/sso/group-mapping/overview.html#supported-cel-operators-for-group-mapping) that defines which groups are included.
- `description` - (Optional String) A description explaining the purpose and use of the group mapping.

## Attributes Reference

In addition to the preceding arguments, the following attributes are exported:

- `id` - (Required String) The ID of the Group Mapping (for example, `group-abc123`).

## Import

-> **Note:** `CONFLUENT_CLOUD_API_KEY` and `CONFLUENT_CLOUD_API_SECRET` environment variables must be set before importing a Group Mapping.

You can import a Group Mapping by using Group Mapping ID, for example:

```shell
$ export CONFLUENT_CLOUD_API_KEY="<cloud_api_key>"
$ export CONFLUENT_CLOUD_API_SECRET="<cloud_api_secret>"
$ terraform import confluent_group_mapping.application-developers group-abc123
```

!> **Warning:** Do not forget to delete terminal command history afterwards for security purposes.
