# Upstreams of the Confluent Terraform Provider

This page lists every service the provider sends requests to, where in the code each connection is made, and how each one retries and times out. It's for maintainers. The customer-facing list of destinations, with proxy and telemetry details, is [docs/guides/network-connections.md](docs/guides/network-connections.md).

When you add, remove or change an upstream (a new SDK client, a new host, a different retry or timeout), update this page and the guide.

Contents:

- [Overview](#overview)
- [Confluent Cloud API clients](#confluent-cloud-api-clients)
- [Cluster data-plane clients](#cluster-data-plane-clients)
- [Other connections](#other-connections)
- [Shared HTTP behavior](#shared-http-behavior)

## Overview

The provider is a plugin process that Terraform starts for one plan, apply, import or destroy. It accepts connections only from Terraform on a local plugin socket. It keeps no cache between runs, and makes these outbound connections:

| Upstream | Destination | Built in | Built when | Retries | Request timeout |
|---|---|---|---|---|---|
| Confluent Cloud API | `endpoint` (default `https://api.confluent.cloud`) | 34 `ccloud-sdk-go-v2` clients in `providerConfigure` ([provider.go](internal/provider/provider.go)) | Once, at Configure | `max_retries` (default 4); at least 12 for the API keys, IAM, RBAC and Connect clients | None |
| STS token exchange | `endpoint` + `/sts/v1/oauth2/token` | `fetchSTSOAuthToken` ([oauth.go](internal/provider/oauth.go)), using the `sts/v1` client | OAuth only: at Configure, and refreshed within 1 minute of expiry | `max_retries` | None |
| Tableflow API | `endpoint` + `/tableflow/v1` | `TableflowRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Kafka REST | Cluster REST endpoint | `KafkaRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Schema Registry | Schema Registry endpoint | `SchemaRegistryRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Stream Catalog | `catalog_rest_endpoint`, else the Schema Registry endpoint | `CatalogRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Flink SQL | Flink regional endpoint | `FlinkRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Customer identity provider | `oauth_external_token_url` | `requestNewExternalOAuthToken` ([oauth.go](internal/provider/oauth.go)) | OAuth with a client ID and secret only: at Configure, and refreshed within 3 minutes of expiry | `max_retries` | None, and the request has no context |
| Cloud storage (S3, Azure Blob, GCS) | Presigned URL returned by a `presigned-upload-url` API | `uploadFile` ([utils.go](internal/provider/utils.go)) | Artifact and plugin creates | None | 20 minutes, and the request has no context |
| Usage telemetry | `endpoint` + `/terraform-usage/v1/usages` | `publishTelemetryRuntime` ([telemetry_runtime.go](internal/provider/telemetry_runtime.go)), [telemetry/](internal/provider/telemetry) | Once, at Configure | None | 5 seconds per event |
| Local `terraform` binary | Subprocess, not a network connection | `commandWithResolvedSymlink` ([resource_tf_importer.go](internal/provider/resource_tf_importer.go)) | `confluent_tf_importer` create | None | The operation's context |

"Request timeout: None" means there's no limit on waiting for a response; see [Shared HTTP behavior](#shared-http-behavior).

## Confluent Cloud API clients

All of these go to `endpoint`. Each client is built once in `providerConfigure` with its own retryable HTTP client, and each has a matching `*ApiContext` helper in [utils.go](internal/provider/utils.go) that adds the Cloud API key or, with OAuth, the STS token. R is a resource and D is a data source, both without the `confluent_` prefix.

| SDK client (`ccloud-sdk-go-v2/…`) | Path prefixes | Used by |
|---|---|---|
| `apikeys/v2` (12+ retries) | `/iam/v2/api-keys` | R api_key |
| `iam/v2` (12+ retries) | `/iam/v2/service-accounts`, `/iam/v2/users`, `/iam/v2/invitations` | R service_account, invitation; D service_account, user, users, invitation |
| `iam/v1` | `/service_accounts`, `/users` (legacy, unversioned) | R tf_importer, to map Kafka ACL principals |
| `mds/v2` (12+ retries) | `/iam/v2/role-bindings` | R/D role_binding |
| `identity-provider/v2` | `/iam/v2/identity-providers` (and identity pools) | R/D identity_provider, identity_pool |
| `certificate-authority/v2` | `/iam/v2/certificate-authorities` (and certificate pools) | R/D certificate_authority, certificate_pool |
| `sso/v2` | `/iam/v2/sso/group-mappings` | R/D group_mapping |
| `iam-ip-filtering/v2` | `/iam/v2/ip-groups`, `/iam/v2/ip-filters` | R/D ip_group, ip_filter |
| `org/v2` | `/org/v2/environments`, `/org/v2/organizations` | R environment; D environment, environments, organization, schema_registry_clusters; also kafka_cluster create, and api_key to check a new Cloud or global key |
| `cmk/v2` | `/cmk/v2/clusters` | R kafka_cluster; D kafka_cluster, kafka_clusters; also api_key, to find a Kafka cluster's REST endpoint |
| `kafka-quotas/v1` | `/kafka-quotas/v1/client-quotas` | R/D kafka_client_quota |
| `srcm/v3` | `/srcm/v3/clusters` | D schema_registry_cluster, schema_registry_clusters; also kafka_cluster create |
| `connect/v1` (12+ retries, jittered 429/503 backoff) | `/connect/v1/environments/…/connectors` | R connector; also tf_importer |
| `connect-custom-plugin/v1` | `/connect/v1/custom-connector-plugins`, `/connect/v1/presigned-upload-url` | R custom_connector_plugin |
| `ccpm/v1` | `/ccpm/v1/plugins`, `/ccpm/v1/presigned-upload-url` | R plugin, custom_connector_plugin_version; D plugin |
| `cam/v1` | `/cam/v1/connect-artifacts`, `/cam/v1/presigned-upload-url` | R/D connect_artifact |
| `flink-artifact/v1` | `/artifact/v1/flink-artifacts`, `/artifact/v1/presigned-upload-url` | R/D flink_artifact |
| `flink/v2` | `/fcpm/v2/compute-pools`, `/fcpm/v2/compute-pool-config`, `/fcpm/v2/regions` | R/D flink_compute_pool, flink_compute_pool_config; D flink_region |
| `ksql/v2` | `/ksqldbcm/v2/clusters` | R/D ksql_cluster |
| `networking/v1` | `/networking/v1/networks`, `peerings`, `transit-gateway-attachments`, `private-link-accesses`, `network-link-services`, `network-link-endpoints`, `network-link-service-associations` | R/D network, peering, transit_gateway_attachment, private_link_access, network_link_service, network_link_endpoint |
| `networking-gateway/v1` | `/networking/v1/gateways` | R/D gateway |
| `networking-access-point/v1` | `/networking/v1/access-points`, `/networking/v1/dns-records` | R/D access_point, dns_record |
| `networking-dnsforwarder/v1` | `/networking/v1/dns-forwarders` | R/D dns_forwarder |
| `networking-ip/v1` | `/networking/v1/ip-addresses` | D ip_addresses |
| `networking-privatelink/v1` | `/networking/v1/private-link-attachments`, `/networking/v1/private-link-attachment-connections` | R/D private_link_attachment, private_link_attachment_connection |
| `endpoint/v1` | `/endpoint/v1/endpoints` | D endpoint |
| `provider-integration/v1`, `provider-integration/v2` | `/pim/v1/integrations`, `/pim/v2/integrations` | R/D provider_integration, provider_integration_setup, provider_integration_authorization |
| `byok/v1` | `/byok/v1/keys` | R/D byok_key |
| `notifications/v1` | `/notifications/v1/…` | R/D notifications_*; D notification_type |
| `rtce/v1` | `/rtce/v1/rtce-topics` | R/D rtce_topic |
| `switchover/v1` | `/switchover/v1/switchover-pairs`, `/switchover/v1/switchover-endpoints` | R/D switchover_pair, switchover_endpoint; D their list forms |
| `sts/v1` | `/sts/v1/oauth2/token` | OAuth token exchange, see [Overview](#overview) |
| `data-catalog/v1` | None | Built at Configure but never called. Stream Catalog calls use `CatalogRestClientFactory` instead. |

The Tableflow API is on the same host but isn't built at Configure: `TableflowRestClientFactory` builds a client per operation, because Tableflow resources can use their own API key. It serves `/tableflow/v1/tableflow-topics`, `/tableflow/v1/catalog-integrations` and `/tableflow/v1/regions` (R/D tableflow_topic, catalog_integration; also api_key, to check a new Tableflow key).

## Cluster data-plane clients

The factories in [factory_utils.go](internal/provider/factory_utils.go) build a new client, with a new retryable HTTP client, for each operation. The host isn't looked up: it comes from the provider block (or its environment variable), the resource's `rest_endpoint` attribute, or an `IMPORT_*` environment variable during import. See the `extract*RestEndpoint` functions, for example `extractRestEndpoint` in [resource_kafka_topic.go](internal/provider/resource_kafka_topic.go).

Authentication is the cluster's API key or, with OAuth, the identity provider's token sent directly (not the STS token), with a `confluent-identity-pool-id` header. The data-plane OAuth refresh helpers are in [utils.go](internal/provider/utils.go) (`kafkaRestApiContextWithClusterApiKey` and the similar functions after it).

| Client | Host from | Path prefixes | Used by |
|---|---|---|---|
| Kafka REST (`kafkarest/v3`) | `kafka_rest_endpoint` (`KAFKA_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_KAFKA_REST_ENDPOINT` | `/kafka/v3/clusters/…` | R kafka_topic, kafka_acl, kafka_cluster_config, cluster_link, kafka_mirror_topic, tf_importer; D kafka_topic, cluster_link; also api_key, to check a new Kafka key for up to 20 minutes, at the REST endpoint read from `/cmk/v2` |
| Schema Registry (`schema-registry/v1`) | `schema_registry_rest_endpoint` (`SCHEMA_REGISTRY_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT` | `/subjects`, `/schemas`, `/config`, `/mode`, `/exporters`, `/dek-registry/v1/keks`, … | R schema, subject_config, subject_mode, schema_registry_cluster_config, schema_registry_cluster_mode, schema_exporter, schema_registry_kek, schema_registry_dek, tf_importer; D the same except schema_exporter and tf_importer, plus schemas. `confluent_schema` also calls it during plan. |
| Stream Catalog (`data-catalog/v1`) | `catalog_rest_endpoint` (`CATALOG_REST_ENDPOINT`), else the Schema Registry endpoint; `rest_endpoint`; `IMPORT_CATALOG_REST_ENDPOINT` | `/catalog/v1/…` | R tag, tag_binding, business_metadata, business_metadata_binding, catalog_entity_attributes; D tag, tag_binding, business_metadata, business_metadata_binding |
| Flink SQL (`flink-gateway/v1`) | `flink_rest_endpoint` (`FLINK_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_FLINK_REST_ENDPOINT` | `/sql/v1/organizations/…` | R flink_statement, flink_connection, flink_materialized_table; D flink_materialized_table |

## Other connections

- **Customer identity provider.** `buildExternalOAuthRequest` ([oauth.go](internal/provider/oauth.go)) POSTs a client-credentials grant to `oauth_external_token_url`, through one retryable client created in `initializeOAuthConfigs` ([provider.go](internal/provider/provider.go)). The request is built with `http.NewRequest`, so it has no context and Ctrl-C doesn't cancel it. It sends Go's default User-Agent. Not called when `oauth_external_access_token` is set.
- **Cloud storage uploads.** `uploadFile` ([utils.go](internal/provider/utils.go)) sends the file to the presigned URL that the `presigned-upload-url` APIs above return: a multipart POST for AWS, and a PUT for Azure (with `x-ms-blob-type: BlockBlob`) and GCP. It uses a plain `http.Client` with a 20-minute timeout, no retries and no context, sends Go's default User-Agent, and doesn't check the response status. Callers: `confluent_flink_artifact`, `confluent_connect_artifact`, `confluent_custom_connector_plugin` and `confluent_custom_connector_plugin_version`.
- **Usage telemetry.** When enabled (see `publishTelemetryRuntime` and the [guide](docs/guides/network-connections.md#usage-telemetry)), each resource operation queues one event, sent by 4 workers from an 8-event queue through the `terraform-usage/v1` SDK with a plain `http.Client`. A full queue or a failed send drops the event, and events still queued when the process exits are lost. With OAuth, the STS token is captured once at Configure.
- **Local `terraform` binary.** `confluent_tf_importer` runs `terraform state replace-provider` on the state it writes, using the `terraform` found on `PATH`. That subprocess makes its own connections, such as Terraform's update check, unless `CHECKPOINT_DISABLE` is set.

The provider makes no other outbound connections. It doesn't check for updates, read feature flags or call metrics APIs. `wiremockTestUtils.go` links testcontainers and the Docker client into the binary, but they're only called from tests.

## Shared HTTP behavior

`RetryableClientFactory.CreateRetryableClient` ([factory_utils.go](internal/provider/factory_utils.go)) builds every SDK and data-plane HTTP client, and the identity provider client:

- **Retries:** [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) `DefaultRetryPolicy` retries connection errors, 429 and 5xx except 501, for every HTTP method. `RetryMax` is `max_retries` (default 4, at least 4); `rateLimitedAPIMaxRetries` raises it to at least 12 for the API keys, IAM, RBAC and Connect clients.
- **Backoff:** `DefaultBackoff`, 1s doubling to at most 30s, no jitter, and on 429 and 503 the `Retry-After` value as-is. The Connect client uses `rateLimitBackoff` instead, which jitters 429/503 waits and uses `Retry-After` as a minimum.
- **Errors:** after the last retry, `customErrorHandler` replaces the response with a message naming the status, URL, method and attempt count.
- **Timeouts:** the transport is go-cleanhttp's `DefaultPooledTransport`: 30s to connect and 10s for the TLS handshake, but no limit on waiting for a response. A request ends only when the operation's context ends: Terraform's default 20 minutes, or the resource's `timeouts` value.
- **Proxies and TLS:** every connection, including the plain `http.Client`s, uses `http.ProxyFromEnvironment` and the system certificate pool.
- **User-Agent:** `buildUserAgent` ([provider.go](internal/provider/provider.go)) builds one User-Agent for all SDK, data-plane and telemetry requests, with the Terraform, plugin SDK and provider versions, plus `TF_APPEND_USER_AGENT` and `user_agent_suffix` if set.
- **Logging:** `loggingTransport` logs each request at debug level.
