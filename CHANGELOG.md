## 1.19.0 (December 1, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.18.0...v1.19.0)

**New features:**
* Added new `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link) and `confluent_kafka_mirror_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_mirror_topic) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Added support for `zones` attribute for `confluent_network` of type `TRANSITGATEWAY`.
* Updated docs ([#150](https://github.com/confluentinc/terraform-provider-confluent/issues/150)).

## 1.18.0 (November 30, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.17.0...v1.18.0)

**New features:**
* Added new `confluent_ksql_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_ksql_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_ksql_cluster) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_region) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added `resource_name` computed attribute to `confluent_ksql_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_ksql_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_ksql_cluster).

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Removed `confluent_stream_governance_region` that was deprecated in `1.16.0` version: The `confluent_stream_governance_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_region) has been removed. Use the `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_region) instead.
* Removed `confluent_stream_governance_cluster` that was deprecated in `1.16.0` version: The `confluent_stream_governance_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_stream_governance_cluster) and [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_cluster) have been removed. Use the `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster) and [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster) instead.

## 1.17.0 (November 29, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.16.0...v1.17.0)

**New features:**
* Added `max_retries` optional attribute (defaults to `4`) for `provider` block to override maximum number of retries for an HTTP client.

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Updated `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link): added new `config` attribute.

## 1.16.0 (November 21, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.15.0...v1.16.0)

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Renamed `confluent_stream_governance_region`: The `confluent_stream_governance_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_region) has been deprecated. Use the `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_region) instead.
* Renamed `confluent_stream_governance_cluster`: The `confluent_stream_governance_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_stream_governance_cluster) and [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_cluster) have been deprecated. Use the `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster) and [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster) instead.
* Follow [Confluent Provider 1.16.0: Upgrade Guide](docs/upgrade-guide-1.16.0.md) to update your TF configuration files accordingly to the renaming changes listed above.

## 1.15.0 (November 18, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.14.0...v1.15.0)

**New features:**
* Added new `confluent_kafka_cluster_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster_config) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#73](https://github.com/confluentinc/terraform-provider-confluent/issues/73)).

**Bug fixes:**
* Fixed "no Kafka ACLs were matched" bug that a user could see when running `terraform plan` after deleting ACLs outside of Terraform ([#141](https://github.com/confluentinc/terraform-provider-confluent/issues/141)).
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Updated `confluent_ksql_cluster`: The `http_endpoint` argument has been removed. Use the `rest_endpoint` argument instead.

## 1.14.0 (November 16, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.13.0...v1.14.0)

**Bug fixes:**
* Added `cleanup.policy` topic setting to list of updatable topic settings. 
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Updated `confluent_ksql_cluster`: The `http_endpoint` argument has been deprecated. Use the `rest_endpoint` argument instead.

## 1.13.0 (November 3, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.12.0...v1.13.0)

**Bug fixes:**
* Updated docs.

## 1.12.0 (November 2, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.11.0...v1.12.0)

**New features:**
* Added new `confluent_stream_governance_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_region) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

## 1.11.0 (October 31, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.10.0...v1.11.0)

**New features:**
* Added new `confluent_transit_gateway_attachment` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_transit_gateway_attachment) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_transit_gateway_attachment) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

## 1.10.0 (October 26, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.9.0...v1.10.0)

**New features:**
* Added new `confluent_stream_governance_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_stream_governance_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_stream_governance_cluster) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#5](https://github.com/confluentinc/terraform-provider-confluent/issues/5)).

**Bug fixes:**
* Increased initial delay when provisioning `confluent_connector` ([#43](https://github.com/confluentinc/terraform-provider-confluent/issues/43)).
* Updated docs.

## 1.9.0 (October 24, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.8.0...v1.9.0)

**New features:**
* Fixed "Error: plugin crashed!" that could be observed when creating instances of `confluent_connector`resource ([#119](https://github.com/confluentinc/terraform-provider-confluent/issues/119)).
* Fixed input validation error for `confluent_cluster_link` resource ([#118](https://github.com/confluentinc/terraform-provider-confluent/issues/118)).
* Updated [dedicated-vpc-peering-aws-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-acls) and [dedicated-vpc-peering-aws-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-rbac) examples to make it possible to run them in a single `terraform apply` step.

**Bug fixes:**
* Updated docs.

## 1.8.0 (October 13, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.7.0...v1.8.0)

**New features:**
* Added new `confluent_kafka_client_quota` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_client_quota) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

## 1.7.0 (October 10, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.6.0...v1.7.0)

**New features:**
* Added new `confluent_kafka_cluster_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster_config) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#73](https://github.com/confluentinc/terraform-provider-confluent/issues/73)).

**Bug fixes:**
* Updated docs.

## 1.6.0 (September 28, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.5.0...v1.6.0)

**New features:**
* Updated `dedicated-privatelink-aws-kafka` and `dedicated-privatelink-azure-kafka` examples to make it possible to run them in a single `terraform apply` step.

**Bug fixes:**
* Updated docs.

## 1.5.0 (September 21, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.4.0...v1.5.0)

**New features:**
* Added new `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link) and `confluent_kafka_mirror_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_mirror_topic) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for updating [Schema Validation topic settings](https://docs.confluent.io/cloud/current/sr/broker-side-schema-validation.html#sv-configuration-options-on-a-topic) for `confluent_kafka_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic).

**Bug fixes:**
* Updated docs.

## 1.4.0 (September 1, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.3.0...v1.4.0)

**New features:**
* Added support for [GCP Private Service Connect](https://cloud.google.com/vpc/docs/private-service-connect) by updating `confluent_network`, `confluent_private_link_access` [resources](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and corresponding [data](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_private_link_access) [sources](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network). See [dedicated-private-service-connect-gcp-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-acls) and [dedicated-private-service-connect-gcp-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-rbac) examples for more details.

## 1.3.0 (August 29, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.2.0...v1.3.0)

**New features:**
* Added new `confluent_ksql_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_ksql_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_ksql_cluster) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

## 1.2.0 (August 18, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.1.0...v1.2.0)

**New features:**
* Added new `confluent_identity_provider` and `confluent_identity_pool` resources and data sources in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Allow users to update the `config_sensitive` attribute for the `confluent_connector` resource ([#84](https://github.com/confluentinc/terraform-provider-confluent/issues/84)).
* Updated docs.

## 1.1.0 (August 9, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.0.0...v1.1.0)

**New features:**
* Released `confluent_connector` resource is now Generally Available and recommended for use in production workflows.

**Bug fixes:**
* Fixed a connector provisioning bug where it was impossible to delete `confluent_connector` via TF if provisioning failed.
* Updated [dedicated-privatelink-aws-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-acls) and [dedicated-privatelink-aws-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-rbac) 
examples to use `zones` attribute of `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) such that created network / Kafka cluster's zones match user VPC's zones ([#80](https://github.com/confluentinc/terraform-provider-confluent/issues/80), [#81](https://github.com/confluentinc/terraform-provider-confluent/issues/81)).
* Updated docs.

## 1.0.0 (June 30, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.13.0...v1.0.0)

[The Confluent Terraform Provider](https://registry.terraform.io/providers/confluentinc/confluent/latest) is now Generally Available and recommended for use in production workflows.

**Bug fixes:**
* Fixed "undefined response type" error for `confluent_connector` resource ([#53](https://github.com/confluentinc/terraform-provider-confluent/issues/53)).
* Updated docs.

## 0.13.0 (June 28, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.12.0...v0.13.0)

**New features**
* Added support for `kafka_api_key`, `kafka_api_secret`, `kafka_rest_endpoint` attributes in a `provider` block to make `rest_endpoint` attribute and `credentials` block optional for `confluent_kafka_acl` and `confluent_kafka_topic` resources ([#37](https://github.com/confluentinc/terraform-provider-confluent/issues/37), [#54](https://github.com/confluentinc/terraform-provider-confluent/issues/54)).
* Added `disable_wait_for_ready` attribute to disable readiness check for `confluent_api_key` resource ([#25](https://github.com/confluentinc/terraform-provider-confluent/issues/25), [#51](https://github.com/confluentinc/terraform-provider-confluent/issues/51)).
* Added support for pausing / resuming a connector by adding `status` attribute for `confluent_connector` resource.

**Bug fixes:**
* Updated docs and added a new [managing-single-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-cluster) example.

## 0.12.0 (June 27, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.11.0...v0.12.0)

**Breaking changes:**
* Reverted resource versioning changes introduced in `0.11.0`. For example, the `confluent_environment_v2` resource was renamed to `confluent_environment`. User feedback on versioned resources made it clear that the pain of manually updating the TF state file outweighs the potential benefits of deprecation flexibility that versioned resources could have provided. In order to avoid forcing users to edit their TF state files (either manually or by running commands like `terraform state mv`) in the future, TF [state migrations](https://www.terraform.io/plugin/sdkv2/resources/state-migration) will be handled within the Confluent Terraform Provider whenever possible.

Follow [Confluent Provider 0.12.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.12.0) to update your TF state and TF configuration files accordingly (direct updates from both [0.10.0](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.12.0#upgrade-guide-upgrading-from-version-0100-of-confluent-provider) and [0.11.0](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.12.0#upgrade-guide-upgrading-from-version-0110-of-confluent-provider) to `0.12.0` are supported).

## 0.11.0 (June 15, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v0.10.0...v0.11.0)

**Breaking changes:**
* Renamed all resources and data sources to contain a version postfix that matches their API group version (find a full list [here](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.11.0)). For example, the `confluent_environment` resource was renamed to `confluent_environment_v2` to match [org/v2](https://docs.confluent.io/cloud/current/api.html#tag/Environments-(orgv2)) API group version.
* Renamed `http_endpoint` attribute to `rest_endpoint` for `confluent_kafka_cluster`, `confluent_kafka_topic`, `confluent_kafka_acl` resources and data sources to match _Cluster settings_ tab on the Confluent Cloud Console where the corresponding attribute is called _REST endpoint_.
* Renamed `api_key` and `api_secret` attributes of `provider` block to `cloud_api_key` and `cloud_api_secret`, respectively.

Follow [Confluent Provider 0.11.0: Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/upgrade-guide-0.11.0) to update your TF state and TF configuration files accordingly.

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
