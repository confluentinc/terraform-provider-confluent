---
page_title: "Supported Examples"
subcategory: ""
description: |-
  An organized index of Terraform configuration examples for the Confluent Cloud Provider.
---

<!--
MAINTENANCE INSTRUCTIONS:
To regenerate this file when new examples are added, use the following Claude prompt:

"Navigate to the subdirectories of examples/configurations/* and rebuild the supported-examples.md file in docs/guides/.

This file should contain an organized index of links to the terraform examples in the directory and subdirectories of https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations.

Each example should have a short description. The index should be grouped logically into categories such as:
- Getting Started
- Dedicated Clusters (by cloud provider and connectivity type)
- Enterprise/Freight Clusters
- Networking
- Security & Authentication
- Schema Registry
- Flink
- Connectors (with subcategories)
- Cluster Links
- Stream Governance & Data Catalog
- ksqlDB
- RBAC & Team Management
- Advanced Features
- Importers & Tools

For directories with subdirectories (like authentication-using-oauth, cluster-link-using-oauth, tableflow, kafka-ops-*-team), include links to each subdirectory example with appropriate subsection headers."

Generated on: 2026-03-10
-->

# Terraform Provider Examples Index

This document provides an organized index of Terraform configuration examples for the Confluent Cloud Provider. All examples are located in the [`examples/configurations`](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations) directory.

## Getting Started

- [**standard-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/standard-kafka-acls) - Basic Standard cluster with Kafka ACLs for authorization
- [**standard-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/standard-kafka-rbac) - Basic Standard cluster with RBAC for authorization
- [**create-default-topic-and-return-kafka-api-keys-to-consume-and-produce**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/create-default-topic-and-return-kafka-api-keys-to-consume-and-produce) - Complete example creating cluster, topic, and API keys for client applications
- [**managing-single-kafka-cluster**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-kafka-cluster) - Manage a single Kafka cluster with topics and ACLs
- [**basic-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls) - Basic Kafka ACL configuration examples
- [**basic-kafka-acls-with-alias**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/basic-kafka-acls-with-alias) - Kafka ACLs using provider aliases for different clusters

## Dedicated Clusters - Public Access

- [**dedicated-public-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-kafka-acls) - Dedicated cluster with public internet access using ACLs
- [**dedicated-public-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-kafka-rbac) - Dedicated cluster with public internet access using RBAC

## Dedicated Clusters - BYOK (Bring Your Own Key)

- [**dedicated-public-aws-byok-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-aws-byok-kafka-acls) - AWS Dedicated cluster with customer-managed encryption keys
- [**dedicated-public-azure-byok-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-azure-byok-kafka-acls) - Azure Dedicated cluster with customer-managed encryption keys
- [**dedicated-public-gcp-byok-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-public-gcp-byok-kafka-acls) - GCP Dedicated cluster with customer-managed encryption keys
- [**azure-key-vault**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/azure-key-vault) - Integration with Azure Key Vault for BYOK
- [**hashicorp-vault**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/hashicorp-vault) - Integration with HashiCorp Vault for secrets management

## Dedicated Clusters - AWS Private Connectivity

- [**dedicated-privatelink-aws-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-acls) - AWS PrivateLink with ACLs
- [**dedicated-privatelink-aws-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-aws-kafka-rbac) - AWS PrivateLink with RBAC
- [**dedicated-transit-gateway-attachment-aws-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-transit-gateway-attachment-aws-kafka-acls) - AWS Transit Gateway attachment with ACLs
- [**dedicated-transit-gateway-attachment-aws-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-transit-gateway-attachment-aws-kafka-rbac) - AWS Transit Gateway attachment with RBAC
- [**dedicated-vpc-peering-aws-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-acls) - AWS VPC Peering with ACLs
- [**dedicated-vpc-peering-aws-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-aws-kafka-rbac) - AWS VPC Peering with RBAC
- [**dedicated-vpc-peering-v2-aws-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-v2-aws-kafka-acls) - AWS VPC Peering v2 with ACLs

## Dedicated Clusters - Azure Private Connectivity

- [**dedicated-privatelink-azure-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-acls) - Azure PrivateLink with ACLs
- [**dedicated-privatelink-azure-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-privatelink-azure-kafka-rbac) - Azure PrivateLink with RBAC
- [**dedicated-vnet-peering-azure-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vnet-peering-azure-kafka-acls) - Azure VNet Peering with ACLs
- [**dedicated-vnet-peering-azure-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vnet-peering-azure-kafka-rbac) - Azure VNet Peering with RBAC

## Dedicated Clusters - GCP Private Connectivity

- [**dedicated-private-service-connect-gcp-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-acls) - GCP Private Service Connect with ACLs
- [**dedicated-private-service-connect-gcp-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-private-service-connect-gcp-kafka-rbac) - GCP Private Service Connect with RBAC
- [**dedicated-vpc-peering-gcp-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-gcp-kafka-acls) - GCP VPC Peering with ACLs
- [**dedicated-vpc-peering-gcp-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dedicated-vpc-peering-gcp-kafka-rbac) - GCP VPC Peering with RBAC

## Enterprise Clusters

- [**enterprise-privatelinkattachment-aws-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/enterprise-privatelinkattachment-aws-kafka-acls) - Enterprise cluster with AWS PrivateLink attachment
- [**enterprise-privatelinkattachment-azure-kafka-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/enterprise-privatelinkattachment-azure-kafka-acls) - Enterprise cluster with Azure PrivateLink attachment
- [**enterprise-pni-aws-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/enterprise-pni-aws-kafka-rbac) - Enterprise cluster with AWS Provider Network Integration

## Freight Clusters

- [**freight-aws-kafka-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/freight-aws-kafka-rbac) - Freight cluster on AWS with RBAC

## Networking

- [**network-access-point-gcp-private-service-connect**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/network-access-point-gcp-private-service-connect) - Network Access Point for GCP Private Service Connect
- [**dns-forwarder**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/dns-forwarder) - DNS forwarder configuration for private networking
- [**provider-integration-aws**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/provider-integration-aws) - AWS provider integration setup

## Security & Authentication

### OAuth Authentication

- [**authentication-using-oauth/azure-entra-id**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth/azure-entra-id) - OAuth authentication with Azure Entra ID (formerly Azure AD)
- [**authentication-using-oauth/okta**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth/okta) - OAuth authentication with Okta
- [**authentication-using-oauth/schema-exporter**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/authentication-using-oauth/schema-exporter) - Schema Exporter with OAuth authentication

### Cluster Link with OAuth

- [**cluster-link-using-oauth/advanced-bidirectional-cluster-link-using-oauth**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cluster-link-using-oauth/advanced-bidirectional-cluster-link-using-oauth) - Advanced bidirectional cluster link with OAuth
- [**cluster-link-using-oauth/destination-initiated-cluster-link-using-oauth**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cluster-link-using-oauth/destination-initiated-cluster-link-using-oauth) - Destination-initiated cluster link with OAuth
- [**cluster-link-using-oauth/regular-bidirectional-cluster-link-using-oauth**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cluster-link-using-oauth/regular-bidirectional-cluster-link-using-oauth) - Regular bidirectional cluster link with OAuth

## Schema Registry

- [**managing-single-schema-registry-cluster**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/managing-single-schema-registry-cluster) - Manage a single Schema Registry cluster
- [**private-link-schema-registry**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/private-link-schema-registry) - Schema Registry with PrivateLink connectivity
- [**single-event-types-avro-schema**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/single-event-types-avro-schema) - Single event type with Avro schema
- [**single-event-types-proto-schema**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/single-event-types-proto-schema) - Single event type with Protobuf schema
- [**single-event-types-proto-schema-with-alias**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/single-event-types-proto-schema-with-alias) - Protobuf schema with provider alias
- [**multiple-event-types-avro-schema**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/multiple-event-types-avro-schema) - Multiple event types with Avro schemas
- [**multiple-event-types-proto-schema**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/multiple-event-types-proto-schema) - Multiple event types with Protobuf schemas
- [**field-level-encryption-schema**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/field-level-encryption-schema) - Field-level encryption for schemas
- [**schema-linking**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/schema-linking) - Schema linking between clusters

## Flink

- [**flink-quickstart**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart) - Quick start guide for Apache Flink on Confluent Cloud
- [**flink-quickstart-2**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-quickstart-2) - Alternative Flink quick start example
- [**private-flink-quickstart**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/private-flink-quickstart) - Flink with private networking
- [**flink_artifact**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink_artifact) - Managing Flink artifacts (UDFs)
- [**flink-connection**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-connection) - Flink connection configuration
- [**flink-carry-over-offset-between-statements**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/flink-carry-over-offset-between-statements) - Managing offsets between Flink statements

## Connectors

- [**connect-artifact**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connect-artifact) - Custom connector plugin management
- [**managed-datagen-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/managed-datagen-source-connector) - Managed Datagen source connector
- [**custom-datagen-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/custom-datagen-source-connector) - Custom Datagen source connector

### Database Connectors

- [**mongo-db-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/mongo-db-source-connector) - MongoDB source connector
- [**mongo-db-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/mongo-db-sink-connector) - MongoDB sink connector
- [**postgre-sql-cdc-debezium-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/postgre-sql-cdc-debezium-source-connector) - PostgreSQL CDC with Debezium
- [**sql-server-cdc-debezium-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/sql-server-cdc-debezium-source-connector) - SQL Server CDC with Debezium

### Cloud Storage Connectors

- [**s3-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/s3-sink-connector) - Amazon S3 sink connector
- [**s3-sink-connector-assume-role**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/s3-sink-connector-assume-role) - S3 sink with IAM role assumption
- [**dynamo-db-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/dynamo-db-sink-connector) - Amazon DynamoDB sink connector

### Data Warehouse Connectors

- [**snowflake-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/snowflake-sink-connector) - Snowflake sink connector
- [**elasticsearch-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/elasticsearch-sink-connector) - Elasticsearch sink connector

### Connector Offset Management

- [**manage-offsets-github-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/manage-offsets-github-source-connector) - Managing offsets for GitHub source connector
- [**manage-offsets-mongo-db-source-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/manage-offsets-mongo-db-source-connector) - Managing offsets for MongoDB source connector
- [**manage-offsets-mysql-sink-connector**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/connectors/manage-offsets-mysql-sink-connector) - Managing offsets for MySQL sink connector

## Cluster Links

- [**source-initiated-cluster-link-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/source-initiated-cluster-link-rbac) - Source-initiated cluster link with RBAC
- [**destination-initiated-cluster-link-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/destination-initiated-cluster-link-rbac) - Destination-initiated cluster link with RBAC
- [**regular-bidirectional-cluster-link-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/regular-bidirectional-cluster-link-rbac) - Bidirectional cluster link with RBAC
- [**advanced-bidirectional-cluster-link-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/advanced-bidirectional-cluster-link-rbac) - Advanced bidirectional cluster link configuration
- [**cluster-link-over-aws-private-link-networks**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cluster-link-over-aws-private-link-networks) - Cluster Link over AWS PrivateLink

## Stream Governance & Data Catalog

- [**stream-catalog**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/stream-catalog) - Stream Catalog (Data Catalog) configuration

### Tableflow

- [**tableflow/byob-aws-storage**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/byob-aws-storage) - Tableflow with Bring Your Own Bucket (BYOB) on AWS
- [**tableflow/confluent-managed-storage**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/confluent-managed-storage) - Tableflow with Confluent-managed storage
- [**tableflow/datagen-connector-byob-aws-storage**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-byob-aws-storage) - Tableflow with Datagen connector using BYOB on AWS
- [**tableflow/datagen-connector-confluent-managed-storage**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/tableflow/datagen-connector-confluent-managed-storage) - Tableflow with Datagen connector using Confluent-managed storage

## ksqlDB

- [**ksql-acls**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-acls) - ksqlDB cluster with ACLs
- [**ksql-rbac**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/ksql-rbac) - ksqlDB cluster with RBAC

## RBAC & Team Management

### Kafka Ops and Environment Admin Teams

- [**kafka-ops-env-admin-product-team/kafka-ops-team**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-env-admin-product-team/kafka-ops-team) - Kafka Ops team configuration
- [**kafka-ops-env-admin-product-team/env-admin-product-team**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-env-admin-product-team/env-admin-product-team) - Environment Admin and Product team configuration

### Kafka Ops and Kafka Admin Teams

- [**kafka-ops-kafka-admin-product-team/kafka-ops-team**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-kafka-admin-product-team/kafka-ops-team) - Kafka Ops team configuration
- [**kafka-ops-kafka-admin-product-team/kafka-admin-product-team**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-ops-kafka-admin-product-team/kafka-admin-product-team) - Kafka Admin and Product team configuration

### Topic as a Service

- [**topic-as-a-service**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/topic-as-a-service) - Topic-as-a-Service pattern with RBAC

## Advanced Features

- [**manage-topics-via-json**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/manage-topics-via-json) - Manage multiple topics using JSON configuration files

## Importers & Tools

- [**cloud-importer**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/cloud-importer) - Import existing Confluent Cloud resources
- [**kafka-importer**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/kafka-importer) - Import existing Kafka resources
- [**schema-registry-importer**](https://github.com/confluentinc/terraform-provider-confluent/tree/master/examples/configurations/schema-registry-importer) - Import existing Schema Registry resources

---

## Additional Resources

- [Terraform Provider Documentation](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs)
- [Sample Project Guide](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/guides/sample-project)
- [Confluent Cloud Documentation](https://docs.confluent.io/cloud/current/overview.html)
