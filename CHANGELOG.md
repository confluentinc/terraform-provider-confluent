## 0.7.0 (May 3, 2022)

Enables automated provisioning with no more manual intervention!

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.6.0...v0.7.0)

**New features**
* Added new resources and corresponding docs:
  * `confluent_api_key` ([#4](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/4), [#17](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/17), [#25](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/25), [#41](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/41), [#66](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/66))
  * `confluent_network` ([#45](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/45))
  * `confluent_peering`
  * `confluent_private_link_access` ([#45](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/45))
* Added new data sources and corresponding docs:
  * `confluent_user` ([#61](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/61))
* Completely rewrote "Sample Project" [guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project) that references 9 TF sample configurations for end-to-end workflows.
* Updated `confluent_kafka_cluster` and `confluent_environment` data sources to accept `display_name` as an input.
* Improved logging to simplify debugging process:
  * Started using `tflog` [package](https://github.com/hashicorp/terraform-plugin-log/tree/main/tflog): now you can [enable detailed logs](https://www.terraform.io/internals/debugging) and use `grep` and a corresponding "logging key" to find all entries related to a particular resource (for example, `grep "environment_id=env-9761j7" log.txt`).
  * Revised and structured logging messages to output non-sensitive attributes instead of unreadable references.
* Added support for [self-managed encryption keys (also known as bring-your-own-key (BYOK) encryption)](https://docs.confluent.io/cloud/current/clusters/byok/index.html). They are only available for Dedicated Kafka clusters on AWS and GCP.

**Bug fixes:**
* Fixed pagination issue for data sources ([#54](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/54), [#68](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/68)).
* Fixed a bug where you could "successfully" import a non-existent resource ([#58](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/58)).
* Fixed a nil pointer exception ([#53](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/53), [#55](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/55), [#67](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/67)).
* Added other minor fixes ([#57](https://github.com/confluentinc/terraform-provider-confluentcloud/issues/57)).

**Breaking changes:**
* All resources and data sources have been renamed in the new [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs). The prefix has changed from `confluentcloud` to `confluent`. For example, `confluentcloud_environment` resource was updated to `confluent_environment`. Please follow the [Confluent Provider 0.7.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.7.0) to update your TF state file.
* Changed `kafka_cluster` attribute type from `string` to `block` for 'confluent_kafka_acl' and 'confluent_kafka_topic' resources and data sources.
* Made `host` attribute required for 'confluent_kafka_acl' resource.

## 0.6.0 (May 3, 2022)

* Deprecated the [Confluent Cloud Terraform Provider](https://github.com/confluentinc/terraform-provider-confluentcloud) in favor of the [Confluent Terraform Provider](https://github.com/confluentinc/terraform-provider-confluent).
