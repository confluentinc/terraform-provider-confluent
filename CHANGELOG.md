## 0.10.0 (June 7, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.9.0...v0.10.0)

**New features**
* Added new `confluent_private_link_access`, `confluent_peering`, `confluent_role_binding` data sources.
* Added more granular examples: [kafka-ops-env-admin-product-team](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-env-admin-product-team) and [kafka-ops-kafka-admin-product-team](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-kafka-admin-product-team).

**Bug fixes:**
* Adjusted waiting time for `confluent_role_binding` resource to avoid sync issues.
* Added client validation for topic name for `confluent_kafka_topic`.
* Resolved 4 Dependabot alerts.
* Update SDK for API Key Mgmt API to display more descriptive errors for `confluent_api_key`.
* Fixed importing error for `confluent_connector`.
* Fixed provisioning error for `confluent_connector` resource ([#43](https://github.com/confluentinc/terraform-provider-confluent/issues/43)).
* Fixed minor documentation issues.

## 0.9.0 (May 25, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.8.0...v0.9.0)

**New features**
* Added new `confluent_network` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) ([#39](https://github.com/confluentinc/terraform-provider-confluent/issues/39)).
* Added `dns_domain` and `zonal_subdomains` computed attributes for `confluent_network` resource ([#40](https://github.com/confluentinc/terraform-provider-confluent/issues/40)).
* Decreased the creation time of `confluent_role_binding` resource by 4.5x ([#24](https://github.com/confluentinc/terraform-provider-confluent/issues/24)).

**Bug fixes:**
* Fixed provisioning error for `confluent_connector` resource ([#43](https://github.com/confluentinc/terraform-provider-confluent/issues/43)).
* Fixed minor documentation issues ([#31](https://github.com/confluentinc/terraform-provider-confluent/issues/31), [#36](https://github.com/confluentinc/terraform-provider-confluent/issues/36)).

## 0.8.0 (May 12, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.7.0...v0.8.0)

**New features**
* Added new `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector) ([#6](https://github.com/confluentinc/terraform-provider-confluent/issues/6)).
* Added new `confluent_organization` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_organization) ([#20](https://github.com/confluentinc/terraform-provider-confluent/issues/20)).
* [Implemented](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_api_key#import) `import` for `confluent_api_key` resource ([#17](https://github.com/confluentinc/terraform-provider-confluent/issues/17)).

**Bug fixes:**
* Updated input validation for `confluent_private_link_access` and `confluent_kafka_cluster` resources ([#18](https://github.com/confluentinc/terraform-provider-confluent/issues/18)).
* Fixed minor documentation issues ([#15](https://github.com/confluentinc/terraform-provider-confluent/issues/15)).

## 0.7.0 (May 3, 2022)

Enables fully automated provisioning with no more manual intervention!

This new Provider ([Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs)) is an important step toward providing a unified experience for provisioning Confluent Cloud and Confluent Platform resources. Follow the [Confluent Provider 0.7.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.7.0) to upgrade from version `0.5.0` of the [Confluent Cloud Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluentcloud/latest/docs) to version `0.7.0` of the [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs).

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
* All resources and data sources have been renamed in the new [Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs). The prefix has been changed from `confluentcloud` to `confluent`. For example, the `confluentcloud_environment` resource was updated to `confluent_environment`. Please follow the [Confluent Provider 0.7.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.7.0) to update your TF state file.
* Changed `kafka_cluster` attribute type from `string` to `block` for 'confluent_kafka_acl' and 'confluent_kafka_topic' resources and data sources.
* Made `host` attribute required for 'confluent_kafka_acl' resource.

## 0.6.0 (May 3, 2022)

* Deprecated the [Confluent Cloud Terraform Provider](https://github.com/confluentinc/terraform-provider-confluentcloud) in favor of the [Confluent Terraform Provider](https://github.com/confluentinc/terraform-provider-confluent).
