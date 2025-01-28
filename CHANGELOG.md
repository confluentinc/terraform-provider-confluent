## 2.14.0 (January 28th, 2025)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.13.0...v2.14.0)

**New features:**
* Updated `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_artifact) to deprecate the `class` attribute and add the `documentation_link` attribute.
The `class` attribute be removed in the next major version of the provider (3.0.0). Refer to the [Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-3-upgrade) for more details.

**Bug fixes:**
* Fixed "Unable to create API key with managed_resource block" issue ([#538](https://github.com/confluentinc/terraform-provider-confluent/issues/538)).

## 2.13.0 (January 23rd, 2025)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.12.0...v2.13.0)

**New features:**
* Updated `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key) to support Tableflow API Keys.
* Added support for resolving private DNS names from a DNS resolver within your own Google Cloud VPC via DNS forwarding. This feature enables fully-managed connectors to access endpoints using private DNS zones. For details, see [DNS forwarding for Google Cloud Peering](https://docs.confluent.io/cloud/current/networking/peering/gcp-peering.html#dns-forwarding-gcp-peering).
* Added support for outbound Google Cloud Private Service Connect connections using Egress Private Service Connect Endpoints. Egress Private Service Connect Endpoints enable fully-managed Confluent connectors to access services from GCP Private Link Service providers such as Google, MongoDB, Snowflake, and others. 
With this capability, Confluent Cloud now supports private outbound connections for Dedicated clusters across all three cloud providers, AWS, Azure, and Google Cloud. For details, see [Google Cloud Egress Private Service Connect Endpoints for Dedicated Clusters](https://docs.confluent.io/cloud/current/networking/gcp-egress-psc.html#cloud-networking-gcp-psc-egress).

**Bug fixes:**
* Resolved an issue with `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact) during the creation of the Presigned URL phase.
* Fixed the "404 error in re-deploying schemas" issue ([#296](https://github.com/confluentinc/terraform-provider-confluent/issues/296)).
* Updated docs ([#506](https://github.com/confluentinc/terraform-provider-confluent/issues/506)).
* Resolved 1 Dependabot alert.

## 2.12.0 (December 3rd, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.11.0...v2.12.0)

**Bug fixes:**
* Optimized fetching mechanism for `confluent_service_account` [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_service_account).

## 2.11.0 (November 22nd, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.10.0...v2.11.0)

**New features:**
* Added support to resume `confluent_flink_statement` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_statement) using different `principal` and/or under a different `compute_pool`.

**Examples:**
* Updated **1** example to explicitly select fields for Flink insert statement from query schema to sink schema to avoid field mismatch:
  * [populated-orders-source-table](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart/statements/populate-orders-source-table.sql).

## 2.10.0 (November 19th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.9.0...v2.10.0)

**New features:**
* Added environment as a required field for `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_artifact) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for `compatibility_group` property for `confluent_schema_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_subject_config) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_subject_config), and for confluent_schema_registry_cluster_config [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster_config) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster_config).
* Added support for Private Network Interface for `confluent_access_point` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_access_point) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_access_point), and `confluent_gateway` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_gateway) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_gateway).
* Released `confluent_certificate_authority` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_certificate_authority) and  [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_certificate_authority), and `confluent_certificate_pool` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_certificate_pool) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_certificate_pool) as General Availability.

**Examples:**
* Added **1** example:
  * [s3-sink-connector-assume-role](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/s3-sink-connector-assume-role).

**Bug fixes:**
* Fix link in Flink statement docs.

## 2.9.0 (October 29th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.8.0...v2.9.0)

**New features:**
* Added new `confluent_gateway` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_gateway) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 2.8.0 (October 25th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.7.0...v2.8.0)

**New features:**
* Added support for the new `latest_offsets` and `latest_offsets_timestamp` attributes of `confluent_flink_statement` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_statement) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for the new `versions` block of `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact) and [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_artifact) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Examples:**
* Added **1** example:
  * [flink-carry-over-offset-between-statements](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-carry-over-offset-between-statements)
* Updated **1** example with more detailed instructions for clarity:
  * [flink-artifact](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink_artifact)

**Bug fixes:**
* Updated docs.

## 2.7.0 (October 17th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.6.0...v2.7.0)

**Bug fixes:**
* Implemented a follow-up to our previous fix for the SR exporter to further prevent provisioning issues.

## 2.6.0 (October 16th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.5.0...v2.6.0)

**New features:**
* Added `confluent_flink_artifact` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_artifact) and [data-source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_artifact) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added `confluent_schema_registry_dek` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_dek), `confluent_schema_registry_kek` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_kek) and corresponding data sources in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added optional `ruleset` and `metadata` blocks, and the `disabled` attribute to the `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Examples:**
* Added **1** example:
  * [flink-artifact](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink_artifact)
* Updated **1** example with more detailed instructions for clarity:
  * [field-level-encryption-schema](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/field-level-encryption-schema)

**Bug fixes:**
* Fixed an issue with stopping Flink statements.
* Extended the default creation timeout for SR exporters to prevent provisioning issues.

## 2.5.0 (October 4th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.4.0...v2.5.0)

**New features:**
* Added support for `confluent_certificate_authority` [resource](docs/resources/confluent_certificate_authority.md) and [data-source](docs/data-sources/confluent_certificate_authority.md) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for `confluent_certificate_pool` [resource](docs/resources/confluent_certificate_pool.md) and [data-source](docs/data-sources/confluent_certificate_pool.md) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 2.4.0 (October 1st, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.3.0...v2.4.0)

**New features:**
* Added support for `confluent_flink_statement` [resource](docs/resources/confluent_flink_statement.md) resume feature.

**Bug fixes:**
* Updated docs for `confluent_kafka_client_quota` [resource](docs/resources/confluent_kafka_client_quota.md) and [data-source](docs/data-sources/confluent_kafka_client_quota.md).

## 2.3.0 (September 20th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.2.0...v2.3.0)

**New features:**
* Added support for `confluent_provider_integration` [resource](docs/resources/confluent_provider_integration.md) and [data-sources](docs/data-sources/confluent_provider_integration.md)

**Bug fixes:**
* Fixed one incorrect docs link in `CHANGELOG.md` for v2.2.0 release.

## 2.2.0 (September 6th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.1.0...v2.2.0)

**New features:**
* Support Private Link for Schema Registry Cluster [data-source](docs/data-sources/confluent_schema_registry_cluster.md).

**Bug fixes:**
* Update docs for [private_link_attachment](docs/resources/confluent_private_link_attachment.md) / [private_link_attachment_connection](docs/resources/confluent_private_link_attachment_connection.md) to indicate that `display_name` is required.

## 2.1.0 (August 26th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v2.0.0...v2.1.0)

**New features:**
* Add `private_endpoint_custom_dns_config_domains` to `network_access_point` [resource](docs/resources/confluent_access_point.md) and [data-source](docs/data-sources/confluent_access_point.md).

**Bug fixes:**
* Update [confluent_catalog_entity_attributes](docs/resources/confluent_catalog_entity_attributes.md) resource docs.
* Update the timeout for `confluent_flink_statement` resource to resolve timeout issues.
* Fix the `confluent_api_key` creation failure ([#418](https://github.com/confluentinc/terraform-provider-confluent/issues/418)).

## 2.0.0 (August 14th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.83.0...v2.0.0)

**New features:**
* Removed deprecated `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_region)
and `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster). Refer to the [Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade) for more details.

**Bug fixes:**
* Update Terraform configuration examples note for [dedicated-vnet-peering-azure-kafka-acls](examples/configurations/dedicated-vnet-peering-azure-kafka-acls/README.md).
* Update Terraform configuration examples note for [dedicated-vnet-peering-azure-kafka-rbac](examples/configurations/dedicated-vnet-peering-azure-kafka-rbac/README.md).

## 1.83.0 (August 7th, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.82.0...v1.83.0)

**Bug fixes:**
* Resolved the Dependabot [alert#47](https://github.com/confluentinc/terraform-provider-confluent/security/dependabot/47).
* Updated [Confluent schema resource document](docs/resources/confluent_schema.md) docs.
* Updated [Confluent Kafka ACL](docs/resources/confluent_kafka_acl.md) docs.
* Added an end-to-end [example](examples/configurations/private-flink-quickstart/main.tf) for Flink with private networking.

## 1.82.0 (July 23, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.81.0...v1.82.0)

* Updated the deprecation message for the `encryption_key` attribute in the `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster). It will continue to exist as a read-only attribute in the next major version of the provider (2.0.0). For more details, refer to the [Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade).
* Updated the `flink-quickstart` example to resolve the "Number of columns must match number of query columns" error.
* Fixed the "Value for unconfigurable attribute" error in some examples.
* Refactored examples to eliminate the use of `local.hosted_zone`.
* Updated the documentation to include an example of creating a role binding for group mapping.
* Updated the documentation to include an updated import command for the `confluent_schema_exporter` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter).

## 1.81.0 (July 17, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.80.0...v1.81.0)

**New features:**
* Added additional validation for `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) ([#292](https://github.com/confluentinc/terraform-provider-confluent/issues/279)).
* Enabled removing accepted invitations from TF State for `confluent_invitation` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_invitation)([#263](https://github.com/confluentinc/terraform-provider-confluent/issues/263)).
* Used FlinkDeveloper in `flink-quickstart` example.

**Bug fixes:**
* Fixed an issue with flooding the TF logs with debug level messages from the SDK ([#372](https://github.com/confluentinc/terraform-provider-confluent/issues/372)).
* Fixed import issues for `confluent_schema_exporter` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter).
* Fixed "schema validation error starting from version 1.70.0" issue ([#378](https://github.com/confluentinc/terraform-provider-confluent/issues/378)).

## 1.80.0 (July 8, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.79.0...v1.80.0)

**New features:**
* Support new `private_rest_endpoint` attribute for the `confluent_flink_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_region).

**Bug fixes:**
* Updated the code of the `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) to make it wait for SR cluster provisioning, 
when the Stream Governance package is selected for the `confluent_environment` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_environment).
* Updated examples to use `confluent_schema_registry_cluster` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster) instead of deprecated [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster).
* Updated docs.

## 1.79.0 (June 26, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.78.0...v1.79.0)

**Bug fixes:**
* Deprecated the `encryption_key` attribute in the `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster). It will be removed in the next major version of the provider (2.0.0). Refer to the [Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade) for more details.
* Update Terraform docs to use `confluent_schema_registry_cluster` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster) instead of deprecated [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster).
* Fixed the "(schema-registry): Invalid Version Exception: 0" issue [#286](https://github.com/confluentinc/terraform-provider-confluent/issues/286) by adding the `skip_validation_during_plan` attribute, which allows users to skip schema validation in the `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).
* Updated the macOS binary's name (with FIPS-140 support) to resolve the "assert does not match provider 'confluent'" error on the HashiCorp Terraform Registry.

## 1.78.0 (June 13, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.77.0...v1.78.0)

**New features:**
* Release an additional macOS binary with FIPS-140 support.

## 1.77.0 (June 10, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.76.0...v1.77.0)

**New features:**
* Added support for the new `freight` Kafka cluster type in the `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Fixed an issue with creating the `confluent_business_metadata_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_business_metadata_binding) ([#362](https://github.com/confluentinc/terraform-provider-confluent/issues/362)).
* Fixed an issue where certain internal configs for the connector were accidentally displayed ([#389](https://github.com/confluentinc/terraform-provider-confluent/issues/389)).
* Fixed an issue where lowering Kafka topic partition count used to cause failure, now it forces topic recreation instead([#383](https://github.com/confluentinc/terraform-provider-confluent/issues/383)).

## 1.76.0 (May 9, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.75.0...v1.76.0)

**New features:**
* Added support for new optional `azure_egress_private_link_endpoint` block of `confluent_access_point` resource and corresponding data source in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for new optional `azure_egress_private_link_gateway` block of `confluent_gateway` data source in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 1.75.0 (May 8, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.74.0...v1.75.0)

**New features:**
* Added new `confluent_catalog_entity_attributes` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_catalog_entity_attributes) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Support new `LOW`, `HIGH` values for the `availability` attribute of the `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster).
* Resolved 1 Dependabot alert.
* Updated docs.

## 1.74.0 (April 29, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.73.0...v1.74.0)

**New features:**
* Support new `cloud` attribute for the `confluent_custom_connector_plugin` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_custom_connector_plugin).

**Bug fixes:**
* Remove principal validation for the `confluent_kafka_acl` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_acl).
* Add waiting method when provisioning the `confluent_dns_forwarder` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_dns_forwarder).

## 1.73.0 (April 23, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.72.0...v1.73.0)

**Bug fixes:**
* Updated `confluent_flink_statement` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_statement) to prevent errors during `terraform apply` if the statement was stopped with a different client.
* Resolved 1 Dependabot alert.
* Updated docs.

## 1.72.0 (April 18, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.71.0...v1.72.0)

**Bug fixes:**
* Resolved "Deprecated schema_registry_region without an alternative" issue ([#373](https://github.com/confluentinc/terraform-provider-confluent/issues/373)).
* Updated docs.

## 1.71.0 (April 11, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.70.0...v1.71.0)

**New features:**
* Added support for new optional `azure` block of `confluent_private_link_attachment`, `confluent_private_link_attachment_connection` resources and corresponding data sources in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **1** new example:
  * [enterprise-privatelinkattachment-azure-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/enterprise-privatelinkattachment-azure-kafka-acls)


## 1.70.0 (April 9, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.68.0...v1.70.0)

**Bug fixes:**
* Resolved "schema content is not updated in state file during refresh" issue ([#318](https://github.com/confluentinc/terraform-provider-confluent/issues/318)). Thanks to @Noel-Jones for contributing the fix!
* Resolved "confluent_schema Resource import should not require SCHEMA_CONTENT" issue ([#293](https://github.com/confluentinc/terraform-provider-confluent/issues/293)). Thanks to @Noel-Jones for contributing the fix!
* Updated docs.

## 1.68.0 (March 29, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.67.0...v1.68.0)

**New features:**
* Added new `confluent_access_point` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_access_point), `confluent_dns_record` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_dns_record) and corresponding data sources in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_gateway` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_gateway) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Updated [flink-quickstart](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart) to use mock data streams.

**Bug fixes:**
* Updated docs.

## 1.67.0 (March 15, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.66.0...v1.67.0)

**New features:**
* Added new `confluent_flink_statement` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_statement) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_flink_compute_pool` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_compute_pool) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_compute_pool) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Increased the timeout for the `confluent_schema_exporter` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter) to resolve potential provisioning issues.
* Resolved 1 Dependabot alert.
* Updated docs.

## 1.66.0 (March 11, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.65.0...v1.66.0)

**New features:**
* Added new `confluent_schema_registry_dek` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_dek), `confluent_schema_registry_kek` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_kek) and corresponding data sources in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new optional `ruleset` and `metadata` blocks to `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**New updates for resources that are in [Early Access / Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** The environment variables for the provider block's `organization_id` and `environment_id` attributes have been renamed from `ORGANIZATION_ID` to `CONFLUENT_ORGANIZATION_ID` and from `ENVIRONMENT_ID` to `CONFLUENT_ENVIRONMENT_ID`, respectively.

**Bug fixes:**
* Updated [flink-quickstart](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart) to use `FlinkAdmin` [role](https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#flinkadmin) instead of `FlinkDeveloper` [role](https://docs.confluent.io/cloud/current/access-management/access-control/rbac/predefined-rbac-roles.html#flinkdeveloper) to be able to delete statements.
* Resolved "Azure Devops Collision with ENVIRONMENT_ID" issue ([#369](https://github.com/confluentinc/terraform-provider-confluent/issues/369)).
* Updated docs.

* Added **1** new example:
  * [field-level-encryption-schema](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/field-level-encryption-schema)

## 1.65.0 (March 4, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.64.0...v1.65.0)

**New updates for resources that are in [Early Access / Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** [Flink API Key Management API](https://docs.confluent.io/cloud/current/api.html) has been updated to utilize a new request format. Ensure you to update to version `1.65.0` to create Flink API Keys using the `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key#example-flink-api-key).

**Bug fixes:**
* Released an updated fix for `CVE-2023-49569`.

## 1.64.0 (February 29, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.63.0...v1.64.0)

**New features:**
* Deprecated `confluent_schema_registry_region` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_region)
and `confluent_schema_registry_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster). They will be
removed in the next major version of the provider (2.0.0). Refer to the [Upgrade Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade) for more details.
  * Note: The [guide]((https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/version-2-upgrade)) will be published once version 2.0.0 is released.

**New updates for resources that are in [Early Access / Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_flink_compute_pool`: The `rest_endpoint` and `current_cfu` deprecated attributes were removed.

## 1.63.0 (February 26, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.62.0...v1.63.0)

**New features:**
* Added new `confluent_dns_forwarder` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_dns_forwarder) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Resolved `CVE-2023-49569`.
* Updated docs.

## 1.62.0 (February 20, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.61.0...v1.62.0)

**New features:**
* Added new `confluent_dns_forwarder` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_dns_forwarder) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `gateway` computed block to `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network).

**New updates for resources that are in [Early Access / Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_flink_statement`: The `organization` and `environment` required blocks were added, the `resource_version` computed attribute was removed. See updated [flink-quickstart](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart) example for more details.
* Updated `confluent_flink_compute_pool`: The `rest_endpoint` and `current_cfu` attributes were deprecated.

**Bug fixes:**
* Resolved 1 Dependabot alert.

## 1.61.0 (January 25, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.60.0...v1.61.0)

**New features:**
* Added new `confluent_users` [data_source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_users) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_registry_clusters` [data_source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_clusters) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_environments` [data_source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_environments) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Fixed "confluent_tag_binding resource should support entity_name updates for sr_record and sr_field" issue ([#348](https://github.com/confluentinc/terraform-provider-confluent/issues/342)).

## 1.60.0 (January 22, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.59.0...v1.60.0)

**Bug fixes:**
* Fixed "instance lsrc-abc123:.:100002 already is associated with classification dynamic" issue when updating `confluent_tag_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tag_binding).
* Fixed "Schema not found when importing a schema in a 20000+ schemas in Schema Registry" issue ([#342](https://github.com/confluentinc/terraform-provider-confluent/issues/342)).

## 1.59.0 (January 12, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.58.0...v1.59.0)

**Bug fixes:**
* Updated a client side fix for "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).

## 1.58.0 (January 11, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.57.0...v1.58.0)

**Bug fixes:**
* Added support for updating credentials for `confluent_business_metadata` and `confluent_business_metadata_binding` resources.
* Updated client side validation rules for `confluent_flink_compute_pool` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_compute_pool).
* Added a client side fix for "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).
* Updated docs.

## 1.57.0 (January 4, 2024)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.56.0...v1.57.0)

**Bug fixes:**
* Added a client side fix for "Add support for new topic configuration fields" issue ([#341](https://github.com/confluentinc/terraform-provider-confluent/issues/341)).
* Added a client side fix for "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).
* Resolved 3 Dependabot alerts.
* Updated docs.

## 1.56.0 (December 12, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.55.0...v1.56.0)

**New features:**
* Added new `confluent_flink_statement` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_statement) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_exporter` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for [custom connectors](https://docs.confluent.io/cloud/current/connectors/bring-your-connector/custom-connector-qs.html) to `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Updated `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key) to support Flink API Keys.
* Added GCP support for `confluent_byok_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_byok_key) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_byok_key) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#65](https://github.com/confluentinc/terraform-provider-confluent/issues/65)).
* Added new `confluent_group_mapping` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_group_mapping) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_group_mapping) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#338](https://github.com/confluentinc/terraform-provider-confluent/issues/338)).
* Added **2** new examples:
  * [flink-quickstart](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart)
  * [dedicated-public-gcp-byok-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-gcp-byok-kafka-acls)

**Bug fixes:**
* Added a client side fix for "confluent_schema_exporter: optional context_type must be set" issue ([#316](https://github.com/confluentinc/terraform-provider-confluent/issues/316)).
* Updated docs.

## 1.55.0 (October 25, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.54.0...v1.55.0)

**New features:**
* Added support for [custom connectors](https://docs.confluent.io/cloud/current/connectors/bring-your-connector/custom-connector-qs.html) to `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector) in [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#243](https://github.com/confluentinc/terraform-provider-confluent/issues/243)).
* Added **1** new example:
  * [custom-datagen-source-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/custom-datagen-source-connector)
* Added new `confluent_ip_addresses` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_ip_addresses) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#300](https://github.com/confluentinc/terraform-provider-confluent/issues/300)).

**Bug fixes:**
* Added a client side fix for "Flink compute pool recreation" issue ([#324](https://github.com/confluentinc/terraform-provider-confluent/issues/324)).
* Added a doc rendering fix from ([PR #312](https://github.com/confluentinc/terraform-provider-confluent/pull/312)).


## 1.54.0 (September 26, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.53.0...v1.54.0)

**New features:**
* Added new `confluent_flink_compute_pool` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_compute_pool) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_compute_pool) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_private_link_attachment` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_private_link_attachment) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_private_link_attachment) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_private_link_attachment_connection` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_private_link_attachment_connection) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_private_link_attachment_connection) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for new optional `enterprise` block of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_cluster) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 1.53.0 (September 21, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.52.0...v1.53.0)

**New features:**
* Added new `confluent_flink_compute_pool` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_flink_compute_pool) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_flink_compute_pool) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 1.52.0 (September 18, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.51.0...v1.52.0)

**New features:**
* Added new `confluent_schema_exporter` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_exporter) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for new _bidirectional_ mode for `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy)

**Bug fixes:**
* Mitigated "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).

## 1.51.0 (July 31, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.50.0...v1.51.0)

**New features:**
* Added new `confluent_private_link_attachment` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_private_link_attachment) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_private_link_attachment) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_private_link_attachment_connection` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_private_link_attachment_connection) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_private_link_attachment_connection) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for new optional `enterprise` block of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_cluster) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **1** new example:
  * [enterprise-privatelinkattachment-aws-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/enterprise-privatelinkattachment-aws-kafka-acls)

**Bug fixes:**
* Fixed "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).

## 1.50.0 (July 21, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.49.0...v1.50.0)

**New features:**
* Added support for new _bidirectional_ mode for `confluent_cluster_link` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_cluster_link) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) with **2** new examples:
  * [`regular-bidirectional-cluster-link-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/regular-bidirectional-cluster-link-rbac): An example of setting up a bidirectional cluster link with 2 mirror topics
  * [`advanced-bidirectional-cluster-link-rbac`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/regular-bidirectional-cluster-link-rbac): An example of setting up a bidirectional cluster link with 2 mirror topics ([advanced option](https://docs.confluent.io/cloud/current/multi-cloud/cluster-linking/cluster-links-cc.html#create-a-cluster-link-in-bidirectional-mode))

**Bug fixes:**
* Fixed "Export max_retries as an environment variable" issue ([#290](https://github.com/confluentinc/terraform-provider-confluent/issues/290)).
* Fixed "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).

## 1.49.0 (July 17, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.48.0...v1.49.0)

**New features:**
* Added new `confluent_schema_registry_clusters` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_clusters) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#254](https://github.com/confluentinc/terraform-provider-confluent/issues/254)).

**Bug fixes:**
* Fixed "Reordering zones shouldn't trigger network recreation" issue ([#288](https://github.com/confluentinc/terraform-provider-confluent/issues/288)).
* Fixed "zones variable in confluent_network resource too restrictive in terms of min/max AZs" issue ([#270](https://github.com/confluentinc/terraform-provider-confluent/issues/270)).
* Fixed "error creating Tag Binding / Business Metadata Binding 404" issue ([#282](https://github.com/confluentinc/terraform-provider-confluent/issues/282)).

## 1.48.0 (July 7, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.47.0...v1.48.0)

**New features:**
* Added new `confluent_environments` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_environments) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#254](https://github.com/confluentinc/terraform-provider-confluent/issues/254)).

## 1.47.0 (June 28, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.46.0...v1.47.0)

**Bug fixes:**
* Updated implementation of `confluent_kafka_acl` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_acl) to fix a rate limiting issue ([#148](https://github.com/confluentinc/terraform-provider-confluent/issues/148)).

## 1.46.0 (June 23, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.45.0...v1.46.0)

**New features:**
* Added new `confluent_network_link_service` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network_link_service) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network_link_service) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_network_link_endpoint` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network_link_endpoint) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network_link_endpoint) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Renamed "Experimental Resource Importer" to ["Resource Importer"](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/resource-importer) and released it in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) enabling import of existing Confluent Cloud resources to Terraform Configuration (`main.tf`) and Terraform State (`terraform.tfstate`) files.
* Added new `confluent_tag` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tag) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tag) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_tag_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tag_binding) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tag_binding) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_business_metadata` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_business_metadata) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_business_metadata) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_business_metadata_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_business_metadata_binding) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_business_metadata_binding) in a [General Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Fixed "TF Resource Importer: Filter out internal topics" issue ([#261](https://github.com/confluentinc/terraform-provider-confluent/issues/261)).
* Fixed "Unexpected behavior for recreate_on_update attribute" issue ([#235](https://github.com/confluentinc/terraform-provider-confluent/issues/235)).
* Updated docs.

## 1.45.0 (June 16, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.44.0...v1.45.0)

**Bug fixes:**
* Fixed "Duplicate resource "confluent_kafka_acl" configuration" bug.
* Fixed "Plugin did not respond" bug ([#258](https://github.com/confluentinc/terraform-provider-confluent/issues/258)).
* Updated docs.

## 1.44.0 (June 15, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.43.0...v1.44.0)

**New features:**
* Added support for `confluent_schema` resource in [Experimental Resource Importer](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/resource-importer).

**Bug fixes:**
* Added missing ACLs in sql-server-cdc-debezium-source-connector example.
* Fixed a bug in the Experimental Resource Importer that occurred when importing resources with the same display name.
* Fixed a bug in the Experimental Resource Importer that occurred when using an API Key with insufficient privileges.
* Fixed the bug that caused the data catalog resources to not be found right after the creation. ([#252](https://github.com/confluentinc/terraform-provider-confluent/issues/252), [#253](https://github.com/confluentinc/terraform-provider-confluent/issues/253))
* Updated docs.

## 1.43.0 (May 31, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.42.0...v1.43.0)

**New features:**
* Added support for `confluent_schema_registry_cluster` resource in [Experimental Resource Importer](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/resource-importer).
* Added support for descriptive validation error messages for `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).

**Bug fixes:**
* Resolved "Apply schema validation during terraform plan phase" issue ([#218](https://github.com/confluentinc/terraform-provider-confluent/issues/218)).
* Resolved "Fix 'no changes' if terraform in-place update failed" issue ([#226](https://github.com/confluentinc/terraform-provider-confluent/issues/226)).
* Resolved "TF Resource Importer: Make output path configurable" issue ([#260](https://github.com/confluentinc/terraform-provider-confluent/issues/260)).
* Resolved "Additional checks in terraform plan" issue ([#224](https://github.com/confluentinc/terraform-provider-confluent/issues/224)) for `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).
* Updated docs.

## 1.42.0 (May 9, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.41.0...v1.42.0)

**New features:**
* Added new `confluent_byok_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_byok_key) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_byok_key) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for new computed `byok_key` block of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_cluster) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Resolved "Support confluent_connector in Experimental Resource Importer" issue ([#248](https://github.com/confluentinc/terraform-provider-confluent/issues/248)).

**Bug fixes:**
* Resolved "Check for correctness of the tag names during terraform plan" issue ([#249](https://github.com/confluentinc/terraform-provider-confluent/issues/249)).
* Resolved "Unable register subject with name containing slashes" issue ([#236](https://github.com/confluentinc/terraform-provider-confluent/issues/236)).
* Updated docs.

## 1.41.0 (May 1, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.40.0...v1.41.0)

**New features:**
* Added new `confluent_tag` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tag) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tag) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_tag_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_tag_binding) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_tag_binding) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_business_metadata` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_business_metadata) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_business_metadata) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_business_metadata_binding` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_business_metadata_binding) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_business_metadata_binding) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **1** new example:
  * [stream-catalog](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/stream-catalog)

**Bug fixes:**
* Updated docs.

## 1.40.0 (April 26, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.39.0...v1.40.0)

**New features:**
* Updated Go version to `1.20` and recompiled binaries for `linux/amd64` and `linux/arm64` to use BoringCrypto library.

**Bug fixes:**
* Resolved "confluent_kafka_cluster is not recreated when type is changed from standard to dedicated" issue ([#221](https://github.com/confluentinc/terraform-provider-confluent/issues/221)).
* Resolved "Fix a minor error in the example to create a confluent_ksql_cluster resource" issue ([#239](https://github.com/confluentinc/terraform-provider-confluent/issues/239)).
* Resolved "Setup Visual Studio Dev Containers to be more easy to develop the module" issue ([#107](https://github.com/confluentinc/terraform-provider-confluent/issues/107)).
* Updated docs.

## 1.39.0 (April 4, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.38.0...v1.39.0)

**New features:**
* Added new [Experimental Resource Importer](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/resource-importer) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) that enables importing your existing Confluent Cloud resources to Terraform Configuration (`main.tf`) and Terraform State (`terraform.tfstate`) files.

## 1.38.0 (March 31, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.37.0...v1.38.0)

**New features:**
* Added new optional `reserved_cidr` attribute and `zone_info` block to `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in a [Limited Availability lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_network_link_service` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network_link_service) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network_link_service) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_network_link_endpoint` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network_link_endpoint) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network_link_endpoint) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **1** new example:
  * [cluster-link-over-aws-private-link-networks](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cluster-link-over-aws-private-link-networks)

## 1.37.0 (March 28, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.36.0...v1.37.0)

**New features:**
* Added new `confluent_invitation` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_invitation) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_invitation) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#133](https://github.com/confluentinc/terraform-provider-confluent/issues/133)).
* Added new `confluent_users` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schemas) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#203](https://github.com/confluentinc/terraform-provider-confluent/issues/203)).
* Added **4** new examples:
  * [azure-key-vault](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/azure-key-vault)
  * [hashicorp-vault](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/hashicorp-vault)
  * [manage-topics-via-json](https://github.com/confluentinc/terraform-provider-confluent-internal/tree/master/examples/configurations/manage-topics-via-json)
  * [topic-as-a-service](https://github.com/confluentinc/terraform-provider-confluent-internal/tree/master/examples/configurations/topic-as-a-service)

**Bug fixes:**
* Fixed a bug "422 Unprocessable Entity: Availability update is only supported on BASIC and STANDARD clusters" when updating `cku` attribute of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster).
* Updated docs.

## 1.36.0 (March 17, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.35.0...v1.36.0)

**New features:**
* Added new `confluent_schemas` [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schemas) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_byok_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_byok_key) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_byok_key) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **2** new examples for `confluent_byok_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_byok_key):
  * [dedicated-public-aws-byok-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-aws-byok-kafka-acls)
  * [dedicated-public-azure-byok-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-azure-byok-kafka-acls)
* Added support for new computed `byok_key` block of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_cluster) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

## 1.35.0 (March 7, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.34.0...v1.35.0)

**New features:**
* Added new `confluent_invitation` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_invitation) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_invitation) in a [Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 1.34.0 (March 1, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.33.0...v1.34.0)

**New features:**
* Added support for new computed `zones` attribute of `confluent_kafka_cluster` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_cluster) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_cluster) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#132](https://github.com/confluentinc/terraform-provider-confluent/issues/132), [#213](https://github.com/confluentinc/terraform-provider-confluent/issues/213)).

**Bug fixes:**
* Updated docs.

## 1.33.0 (February 28, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.32.0...v1.33.0)

**New features:**
* Added support for new optional `dns_config` block of `confluent_network` on Azure and GCP [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_subject_mode` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_subject_mode) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_subject_mode) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#155](https://github.com/confluentinc/terraform-provider-confluent/issues/155)).
* Added new `confluent_subject_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_subject_config) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_subject_config) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_registry_cluster_mode` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster_mode) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster_mode) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#155](https://github.com/confluentinc/terraform-provider-confluent/issues/155)).
* Added new `confluent_schema_registry_cluster_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster_config) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster_config) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.
* Updated [ksql-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-rbac) example to replace `CloudClusterAdmin` role with `ResourceOwner` and `KsqlAdmin` [roles](https://docs.confluent.io/cloud/current/access-management/access-control/cloud-rbac.html#ccloud-rbac-roles).
* Fixed "KsqlAdmin role for ksqldb doesn't work" bug in [ksql-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-acls) example ([#198](https://github.com/confluentinc/terraform-provider-confluent/issues/198)).
* Fixed a bug to display a descriptive error message when updating name of `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector) ([#171](https://github.com/confluentinc/terraform-provider-confluent/issues/171)).
* Fixed a bug to load schemas from in all contexts and not just `default` one to create a unified experience with the Confluent Cloud Console.

## 1.32.0 (February 15, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.31.0...v1.32.0)

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_schema`: Fixed a follow-up issue for "Error customizing diff Schema: 422 Unprocessable Entity" bug ([#196](https://github.com/confluentinc/terraform-provider-confluent/issues/196)). You might have to reimport your existing instances of `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).

## 1.31.0 (February 14, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.30.0...v1.31.0)

**New features:**
* Added support for new optional `dns_config` block of `confluent_network` on Azure and GCP [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy). More specifically, The value `PRIVATE` for `dns_config.resolution` is in [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) for AWS networks with `PRIVATELINK` connection type. It is in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) for GCP and Azure networks with `PRIVATELINK` connection type.

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_schema`: Fixed "Error customizing diff Schema: 422 Unprocessable Entity" bug ([#196](https://github.com/confluentinc/terraform-provider-confluent/issues/196)). You might have to reimport your existing instances of `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).

## 1.30.0 (February 13, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.29.0...v1.30.0)

**New features:**
* Added new optional `reserved_cidr` attribute and `zone_info` block to `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.
* Fixed a bug to allow update references in `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).

## 1.29.0 (February 8, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.28.0...v1.29.0)

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_schema`: Added checks for semantic (rather than syntactic) equivalence of schemas to avoid occasional Terraform drift during schema updates ([#181](https://github.com/confluentinc/terraform-provider-confluent/issues/181)). You should reimport your existing instances of `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema).

## 1.28.0 (January 30, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.27.0...v1.28.0)

**New features:**
* Added new optional `dns_config` block to `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

## 1.27.0 (January 30, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.26.0...v1.27.0)

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* **Breaking changes:** Updated `confluent_schema`: The `recreate_on_update` and `hard_delete` attributes were added. You should reimport your existing instances of `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema) ([#176](https://github.com/confluentinc/terraform-provider-confluent/issues/176), [#179](https://github.com/confluentinc/terraform-provider-confluent/issues/179)).

## 1.26.0 (January 27, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.25.0...v1.26.0)

**New features:**
* Added support for updating the `partitions_count` attribute for `confluent_kafka_topic` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_topic).
* Added **1** new example for `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector):
  * [sql-server-cdc-debezium-source-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/sql-server-cdc-debezium-source-connector)

**Bug fixes:**
* Fixed a typo in docs for `confluent_kafka_client_quota` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_client_quota): `"<default>"` (and not `"default"`) should be used represent the default quota.

## 1.25.0 (January 19, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.24.0...v1.25.0)

**New features:**
* Added new optional `dns_config` block to `confluent_network` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_network) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_network) in an [Early Access lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added **7** new examples for `confluent_connector` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_connector):
  * [s3-sink-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/s3-sink-connector)
  * [snowflake-sink-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/snowflake-sink-connector)
  * [elasticsearch-sink-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/elasticsearch-sink-connector)
  * [dynamo-db-sink-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/dynamo-db-sink-connector)
  * [mongo-db-source-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/mongo-db-source-connector)
  * [mongo-db-sink-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/mongo-db-sink-connector)
  * [postgre-sql-cdc-debezium-source-connector](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/postgre-sql-cdc-debezium-source-connector)

**Bug fixes:**
* Added support for `zones` attribute for `confluent_network` of type `PEERING`.

## 1.24.0 (January 5, 2023)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.23.0...v1.24.0)

**New features:**
* Added new `confluent_subject_mode` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_subject_mode) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_subject_mode) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#155](https://github.com/confluentinc/terraform-provider-confluent/issues/155)).
* Added new `confluent_subject_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_subject_config) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_subject_config) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_schema_registry_cluster_mode` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster_mode) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster_mode) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy) ([#155](https://github.com/confluentinc/terraform-provider-confluent/issues/155)).
* Added new `confluent_schema_registry_cluster_config` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema_registry_cluster_config) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema_registry_cluster_config) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added support for `kafka_id` attribute in the `provider` block ([#37](https://github.com/confluentinc/terraform-provider-confluent/issues/37#issuecomment-1169098579)). See [managing-single-kafka-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster) example for more details.
* Added support for `schema_registry_id` attribute in the `provider` block ([#124](https://github.com/confluentinc/terraform-provider-confluent/issues/124#issuecomment-1339650088)). See [managing-single-schema-registry-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster) example for more details.
* Added new examples:
  * [managing-single-kafka-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster)
  * [managing-single-schema-registry-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster)
  * [basic-kafka-acls-with-alias](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls-with-alias)
  * [single-event-types-proto-schema-with-alias](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/single-event-types-proto-schema-with-alias)

**Bug fixes:**
* Fixed "confluent_kafka_acl resource does not allow use as principal 'User:*'" ([#152](https://github.com/confluentinc/terraform-provider-confluent/issues/152)).
* Resolved 4 Dependabot alerts.
* Fixed a bug in [ksql-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-rbac) example.
* Updated [dedicated-privatelink-azure-kafka-acls](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-acls), [dedicated-privatelink-azure-kafka-rbac](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-rbac) examples to remove check for _disabled [Private Link endpoint network policies](https://docs.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy)_.
* Updated docs ([#160](https://github.com/confluentinc/terraform-provider-confluent/issues/160), [#161](https://github.com/confluentinc/terraform-provider-confluent/issues/161)).

## 1.23.0 (December 16, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.22.0...v1.23.0)

**New features:**
* Updated `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key) to support ksqlDB API Keys.

**Bug fixes:**
* Updated docs.
* Updated examples.

## 1.22.0 (December 15, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.21.0...v1.22.0)

**New features:**
* Added new `confluent_identity_provider` and `confluent_identity_pool` resources and data sources in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Updated `confluent_api_key` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_api_key) to support Schema Registry API Keys.

**Bug fixes:**
* Updated docs.
* Updated examples.

## 1.21.0 (December 8, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.20.0...v1.21.0)

**New features:**
* Added new `confluent_transit_gateway_attachment` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_transit_gateway_attachment) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_transit_gateway_attachment) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).
* Added new `confluent_kafka_client_quota` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_kafka_client_quota) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_kafka_client_quota) in a [Generally Available lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Updated `confluent_transit_gateway_attachment`: The `enable_custom_routes` attribute has been removed. The`routes` attribute is required now.

## 1.20.0 (December 5, 2022)

[Full Changelog](https://github.com/confluentinc/terraform-provider-confluent/compare/v1.19.0...v1.20.0)

**New features:**
* Added new `confluent_schema` [resource](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/confluent_schema) and a corresponding [data source](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/data-sources/confluent_schema) in an [Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy).

**Bug fixes:**
* Updated docs.

**New updates for resources that are in [Early Access / Open Preview lifecycle stage](https://docs.confluent.io/cloud/current/api.html#section/Versioning/API-Lifecycle-Policy):**
* Updated `confluent_transit_gateway_attachment`: The `enable_custom_routes` attribute has been deprecated. The `enable_custom_routes` attribute will be removed in the next release and `routes` attribute will be made required.

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
* Updated docs and added a new [managing-single-kafka-cluster](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster) example.

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
