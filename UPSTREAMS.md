# Upstreams of the Confluent Terraform Provider

This page lists the services the provider sends requests to, where in the code each connection is made, and how each one authenticates, retries and times out. It's for maintainers.

Last checked against the code at v2.94.0. When you add, remove or change an upstream (a new SDK client, a new host, a different retry or timeout), update this page.

Contents:

- [Overview](#overview)
- [Confluent Cloud API clients](#confluent-cloud-api-clients)
- [Cluster data-plane clients](#cluster-data-plane-clients)
- [Other connections](#other-connections)
- [Shared HTTP behavior](#shared-http-behavior)

## Overview

The provider is a plugin process that Terraform starts for each provider configuration in each command phase (plan, apply, import, destroy). It accepts connections only from Terraform, on a local plugin socket (a loopback TCP port on Windows). It keeps no cache between runs, and makes these outbound connections:

| Upstream | Destination | Built in | Built when | Retries | Request timeout |
|---|---|---|---|---|---|
| Confluent Cloud API | `endpoint` (default `https://api.confluent.cloud`) | One `ccloud-sdk-go-v2` client per SDK package, in `providerConfigure` ([provider.go](internal/provider/provider.go)) | Once, at Configure | `max_retries` (default 4); at least 12 for the `apikeys/v2`, `iam/v2`, `mds/v2` and `connect/v1` clients | None |
| STS token exchange | `endpoint` + `/sts/v1/oauth2/token` | `fetchSTSOAuthToken` ([oauth.go](internal/provider/oauth.go)), using the `sts/v1` client | OAuth only: at Configure, then on the first call within 1 minute of expiry (half the lifetime if shorter) | `max_retries` | None |
| Tableflow API | `endpoint` + `/tableflow/v1` | `TableflowRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Kafka REST | Cluster REST endpoint | `KafkaRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Schema Registry | Schema Registry endpoint | `SchemaRegistryRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Stream Catalog | `catalog_rest_endpoint`, else the Schema Registry endpoint | `CatalogRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Flink SQL | Flink regional endpoint | `FlinkRestClientFactory` ([factory_utils.go](internal/provider/factory_utils.go)) | Per operation | `max_retries` | None |
| Customer identity provider | `oauth_external_token_url` | `requestNewExternalOAuthToken` ([oauth.go](internal/provider/oauth.go)) | OAuth with a token URL only: at Configure, then on the first call within 3 minutes of expiry (half the lifetime if shorter) | `max_retries` | None, and the request has no context |
| Cloud storage (S3, Azure Blob, GCS) | Presigned URL returned by a `presigned-upload-url` API | `uploadFile` ([utils.go](internal/provider/utils.go)) | Create of flink_artifact, connect_artifact, custom_connector_plugin, custom_connector_plugin_version | None | 20 minutes, and the request has no context |
| Local `terraform` binary | Subprocess, not a network connection | `commandWithResolvedSymlink` ([resource_tf_importer.go](internal/provider/resource_tf_importer.go)) | `confluent_tf_importer` create | None | The create's context |

"Request timeout: None" means there's no limit on waiting for a response; see [Shared HTTP behavior](#shared-http-behavior).

## Confluent Cloud API clients

`providerConfigure` builds 34 SDK clients, each with its own retryable HTTP client. All except `data-catalog/v1` are pointed at `endpoint`. Each one, except `sts/v1` and `data-catalog/v1`, has a matching `(c *Client) …ApiContext` helper in [utils.go](internal/provider/utils.go) that adds the Cloud API key or, with OAuth, the STS token. Exceptions:

- `cam/v1`: `camV1ApiContext` sends only the Cloud API key (no OAuth branch) and uses `context.WithoutCancel`, so its requests aren't cancelled when the operation ends.
- `ccpm/v1`: `ccpmV1ApiContext` stores the STS token under the `connect-custom-plugin/v1` context key, which the `ccpm/v1` SDK doesn't read, so with OAuth its requests carry no credentials.

R is a resource and D is a data source, both without the `confluent_` prefix.

| SDK client (`ccloud-sdk-go-v2/…`) | Path prefixes | Used by |
|---|---|---|
| `apikeys/v2` (12+ retries) | `/iam/v2/api-keys` | R api_key |
| `iam/v2` (12+ retries) | `/iam/v2/service-accounts`, `/iam/v2/users`, `/iam/v2/invitations` | R service_account, invitation; D service_account, user, users, invitation; also tf_importer |
| `iam/v1` | `/service_accounts`, `/users` (legacy, unversioned) | R tf_importer, to map Kafka ACL principals |
| `mds/v2` (12+ retries) | `/iam/v2/role-bindings` | R/D role_binding |
| `identity-provider/v2` | `/iam/v2/identity-providers` (and identity pools) | R/D identity_provider, identity_pool |
| `certificate-authority/v2` | `/iam/v2/certificate-authorities` (and certificate pools) | R/D certificate_authority, certificate_pool |
| `sso/v2` | `/iam/v2/sso/group-mappings` | R/D group_mapping |
| `iam-ip-filtering/v2` | `/iam/v2/ip-groups`, `/iam/v2/ip-filters` | R/D ip_group, ip_filter |
| `org/v2` | `/org/v2/environments` | R environment; D environment, environments, organization, schema_registry_clusters; also kafka_cluster create, tf_importer, and api_key to check a new Cloud or global key (authenticating with that key) |
| `cmk/v2` | `/cmk/v2/clusters` | R kafka_cluster; D kafka_cluster, kafka_clusters; also tf_importer, and api_key to find a Kafka cluster's REST endpoint |
| `kafka-quotas/v1` | `/kafka-quotas/v1/client-quotas` | R/D kafka_client_quota |
| `srcm/v3` | `/srcm/v3/clusters` | D schema_registry_cluster, schema_registry_clusters; also kafka_cluster create |
| `connect/v1` (12+ retries, jittered 429/503 backoff) | `/connect/v1/environments/…/clusters/…/connectors`, `…/connector-plugins/…/config/validate` | R connector; also tf_importer |
| `connect-custom-plugin/v1` | `/connect/v1/custom-connector-plugins`, `/connect/v1/presigned-upload-url` | R custom_connector_plugin |
| `ccpm/v1` | `/ccpm/v1/plugins`, `/ccpm/v1/presigned-upload-url` | R plugin, custom_connector_plugin_version; D plugin |
| `cam/v1` | `/cam/v1/connect-artifacts`, `/cam/v1/presigned-upload-url` | R/D connect_artifact |
| `flink-artifact/v1` | `/artifact/v1/flink-artifacts`, `/artifact/v1/presigned-upload-url` | R/D flink_artifact |
| `flink/v2` | `/fcpm/v2/compute-pools`, `/fcpm/v2/compute-pool-config`, `/fcpm/v2/regions` | R/D flink_compute_pool, flink_compute_pool_config; D flink_region |
| `ksql/v2` | `/ksqldbcm/v2/clusters` | R/D ksql_cluster |
| `networking/v1` | `/networking/v1/networks`, `peerings`, `transit-gateway-attachments`, `private-link-accesses`, `network-link-services`, `network-link-endpoints` | R/D network, peering, transit_gateway_attachment, private_link_access, network_link_service, network_link_endpoint |
| `networking-gateway/v1` | `/networking/v1/gateways` | R/D gateway; D gateways |
| `networking-access-point/v1` | `/networking/v1/access-points`, `/networking/v1/dns-records` | R/D access_point, dns_record |
| `networking-dnsforwarder/v1` | `/networking/v1/dns-forwarders` | R/D dns_forwarder |
| `networking-ip/v1` | `/networking/v1/ip-addresses` | D ip_addresses |
| `networking-privatelink/v1` | `/networking/v1/private-link-attachments`, `/networking/v1/private-link-attachment-connections` | R/D private_link_attachment, private_link_attachment_connection |
| `endpoint/v1` | `/endpoint/v1/endpoints` | D endpoint |
| `provider-integration/v1` | `/pim/v1/integrations` | R/D provider_integration |
| `provider-integration/v2` | `/pim/v2/integrations` | R/D provider_integration_setup, provider_integration_authorization |
| `byok/v1` | `/byok/v1/keys` | R/D byok_key |
| `notifications/v1` | `/notifications/v1/…` | R/D notifications_*; D notification_type |
| `rtce/v1` | `/rtce/v1/rtce-topics` | R/D rtce_topic |
| `switchover/v1` | `/switchover/v1/switchover-pairs`, `/switchover/v1/switchover-endpoints` | R/D switchover_pair, switchover_endpoint; D their list forms |
| `sts/v1` | `/sts/v1/oauth2/token` | OAuth token exchange, see [Overview](#overview) |
| `data-catalog/v1` | None | Built at Configure (not pointed at `endpoint`) but never called. Stream Catalog calls use `CatalogRestClientFactory` instead. |

The Tableflow API is on the same host but isn't built at Configure: `TableflowRestClientFactory` builds a client per operation, because Tableflow calls always authenticate with a Tableflow API key (from the provider block, the resource's `credentials` block, or `IMPORT_TABLEFLOW_*`), never the Cloud API key or STS token. With OAuth it adds only a `confluent-identity-pool-id` header. It serves `/tableflow/v1/tableflow-topics`, `/tableflow/v1/catalog-integrations` and `/tableflow/v1/regions` (R/D tableflow_topic, catalog_integration; also api_key, to check a new Tableflow key).

## Cluster data-plane clients

The factories in [factory_utils.go](internal/provider/factory_utils.go) build a new client, with a new retryable HTTP client and connection pool, for each operation. The host isn't looked up (except by `confluent_api_key`, below). It comes from the provider block, but only when the block sets the cluster's whole group of settings; otherwise from the resource's `rest_endpoint` attribute, or an `IMPORT_*` environment variable during import. See the `extract*RestEndpoint` functions, for example `extractRestEndpoint` in [resource_kafka_topic.go](internal/provider/resource_kafka_topic.go).

Authentication is the cluster's API key or, with OAuth, the identity provider's token sent directly (not the STS token), with a `confluent-identity-pool-id` header; Schema Registry and Stream Catalog also send `target-sr-cluster`. The OAuth refresh happens in each client's `apiContext` method in [utils.go](internal/provider/utils.go) (`dataCatalogV1ApiContext` for `CatalogRestClient`). A refreshed token is kept only on that per-operation client.

| Client | Host from | Path prefixes | Used by |
|---|---|---|---|
| Kafka REST (`kafkarest/v3`) | `kafka_rest_endpoint` (`KAFKA_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_KAFKA_REST_ENDPOINT`. R cluster_link never uses the provider block: it uses the `rest_endpoint` in its `source_kafka_cluster`, `destination_kafka_cluster` or `local_kafka_cluster` block, or `IMPORT_{SOURCE,DESTINATION,LOCAL}_KAFKA_REST_ENDPOINT` on import. kafka_mirror_topic uses the `rest_endpoint` in its `kafka_cluster` block, except on import. | `/kafka/v3/clusters/…` | R kafka_topic, kafka_acl, kafka_cluster_config, cluster_link, kafka_mirror_topic, tf_importer; D kafka_topic, cluster_link; also api_key, to check a new Kafka key (basic auth, every minute for up to 20 minutes, unless `disable_wait_for_ready`) at the cluster's `http_endpoint` read from `/cmk/v2` |
| Schema Registry (`schema-registry/v1`) | `schema_registry_rest_endpoint` (`SCHEMA_REGISTRY_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT` | `/subjects`, `/schemas`, `/compatibility`, `/config`, `/mode`, `/exporters`, `/dek-registry/v1/keks`, … | R schema, subject_config, subject_mode, schema_registry_cluster_config, schema_registry_cluster_mode, schema_exporter, schema_registry_kek, schema_registry_dek, tf_importer; D the same except schema_exporter and tf_importer, plus schemas. `confluent_schema`'s `CustomizeDiff` also calls it during plan when `schema` changes. |
| Stream Catalog (`data-catalog/v1`) | `catalog_rest_endpoint` (`CATALOG_REST_ENDPOINT`), else the Schema Registry endpoint; `rest_endpoint`; `IMPORT_CATALOG_REST_ENDPOINT`, else `IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT` | `/catalog/v1/…` | R tag, tag_binding, business_metadata, business_metadata_binding, catalog_entity_attributes; D tag, tag_binding, business_metadata, business_metadata_binding |
| Flink SQL (`flink-gateway/v1`) | `flink_rest_endpoint` (`FLINK_REST_ENDPOINT`), `rest_endpoint`, `IMPORT_FLINK_REST_ENDPOINT` | `/sql/v1/organizations/…` | R flink_statement, flink_connection, flink_materialized_table; D flink_connection, flink_materialized_table |

## Other connections

- **Customer identity provider.** `buildExternalOAuthRequest` ([oauth.go](internal/provider/oauth.go)) POSTs a client-credentials grant to `oauth_external_token_url`, through one retryable client created in `initializeOAuthConfigs` ([provider.go](internal/provider/provider.go)). The request is built with `http.NewRequest`, so it has no context and Ctrl-C doesn't cancel it. It sends Go's default User-Agent. Not called when `oauth_external_access_token` is set.
- **Cloud storage uploads.** `uploadFile` ([utils.go](internal/provider/utils.go)) sends the file to the presigned URL that the `presigned-upload-url` APIs above return: a multipart POST for AWS, and a PUT for Azure (with `x-ms-blob-type: BlockBlob`) and GCP. It uses a plain `http.Client` with a 20-minute timeout, no retries and no context, sends Go's default User-Agent, and doesn't check the response status.
- **Local `terraform` binary.** `confluent_tf_importer` runs `terraform state replace-provider` on the state it writes, using the `terraform` found on `PATH`, in the provider's working directory and environment. That subprocess makes its own connections, such as Terraform's update check, unless `CHECKPOINT_DISABLE` is set.

The provider doesn't check for updates, read feature flags or call metrics APIs. `wiremockTestUtils.go` isn't a `_test.go` file, so testcontainers and its dependencies are linked into the binary. Their functions are only called from tests, but their `init()` code runs at every provider start: it lists the host's processes (no network), and on Windows gopsutil also tries to enable `SeDebugPrivilege` for the provider process.

## Shared HTTP behavior

`RetryableClientFactory.CreateRetryableClient` ([factory_utils.go](internal/provider/factory_utils.go)) builds the HTTP client of every Cloud API SDK client built at Configure, of every data-plane client, and of the identity provider requests:

- **Retries:** [go-retryablehttp](https://github.com/hashicorp/go-retryablehttp) `DefaultRetryPolicy` retries connection errors (except TLS certificate errors), 429 and 5xx except 501, for every HTTP method. `RetryMax` is `max_retries` (default 4, at least 4); `rateLimitedAPIMaxRetries` raises it to at least 12 for the `apikeys/v2`, `iam/v2`, `mds/v2` and `connect/v1` clients.
- **Backoff:** `DefaultBackoff`, 1s doubling to at most 30s, no jitter, and on 429 and 503 the `Retry-After` value as-is. The Connect client uses `rateLimitBackoff` instead, which jitters 429/503 waits and uses `Retry-After` as a minimum.
- **Errors:** after the last retry of a 429 or 5xx, `customErrorHandler` turns the response into an error naming the status, URL, method and attempt count (labelled `Retries`). Connection errors are returned as they are.
- **Timeouts:** the transport is go-cleanhttp's `DefaultPooledTransport`: 30s to connect and 10s for the TLS handshake, but no limit on waiting for a response. A request ends when its context ends. For create, read, update, delete and data-source reads, that's the resource's `timeouts` value if set, else its declared default (for example 24 hours for `confluent_connector` create), else the plugin SDK's 20 minutes. Import, plan-time `CustomizeDiff` calls, the STS exchange at Configure and `cam/v1` calls with a Cloud API key have no deadline.
- **Proxies and TLS:** every connection uses a transport with `http.ProxyFromEnvironment` (go-cleanhttp's for the retryable clients, Go's default for the upload's plain `http.Client`), so it honors `HTTPS_PROXY` and `NO_PROXY` and the system certificate pool.
- **User-Agent:** `buildUserAgent` ([provider.go](internal/provider/provider.go)) builds one User-Agent for all SDK and data-plane requests, with the Terraform, plugin SDK and provider versions, plus `TF_APPEND_USER_AGENT` and `user_agent_suffix` if set. Each `confluent_tf_importer` create appends `TFImporter` again to the User-Agent of the 13 Cloud API clients it may use and of the Kafka REST factory, for the rest of the process (`overrideUserAgent`).
- **Logging:** go-retryablehttp's own log lines go to `tflog` through `retryClientLogger` (attempts and retries at debug, connection failures at error). `loggingTransport` also logs, at debug and once per request after retries, the `x-request-id`, method, URL and status of each response that has an `x-request-id`.
