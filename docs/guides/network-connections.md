---
page_title: "Network Connections"
---

# Network Connections Made by the Confluent Provider

This guide lists every destination the Confluent Terraform Provider connects to, so you can set up firewalls, egress allowlists and proxies for the machines that run Terraform.

Terraform runs the provider as a plugin on the same machine or CI runner as `terraform plan` or `terraform apply`, and the provider makes every connection below from there. All of them use HTTPS on port 443, unless a URL you configure names another port.

For most setups, allowing HTTPS to `*.confluent.cloud`, your identity provider (with OAuth) and your cloud's storage service (for file uploads) is enough. Confluent Cloud for Government uses different hostnames and isn't covered here.

Not covered here:

* Connections made by Terraform itself, such as downloading the provider during `terraform init` and reading or writing remote state.
* Connections that Confluent Cloud makes on your behalf, such as cluster links to remote clusters, schema exporters to destination Schema Registries, and connectors to external systems. The provider only sends their settings and credentials to Confluent Cloud, through the destinations below.

## Confluent Cloud API

| Destination | Used for | When | Authentication |
|---|---|---|---|
| `api.confluent.cloud` | Managing control-plane resources, such as environments, Kafka clusters, networks, service accounts, API keys, role bindings, connectors, Flink compute pools, Tableflow topics and catalog integrations | Nearly every resource and data source, during plan, apply, import and destroy | Cloud API key, or with OAuth, a Confluent Cloud token from the token exchange below. Tableflow requests always use a Tableflow API key, even with OAuth. |
| `api.confluent.cloud/sts/v1/oauth2/token` | Exchanging your identity provider's token for a Confluent Cloud token | OAuth only: each time Terraform configures the provider, and again shortly before the Confluent Cloud token expires | Your identity provider's token |
| `api.confluent.cloud/terraform-usage/v1/usages` | Usage telemetry, see [Usage Telemetry](#usage-telemetry) | After each resource create, read, update, delete and import | Cloud API key, or with OAuth, a Confluent Cloud token |

To use a different Confluent Cloud API endpoint, set `endpoint` in the provider block. The API and the token exchange then use that endpoint, and usage telemetry is turned off.

## Confluent Cloud Clusters

The provider doesn't look up these addresses, except for the `confluent_api_key` check noted below. Each one comes from a provider setting (or its environment variable), from the resource's `rest_endpoint` attribute, or, during `terraform import`, from an `IMPORT_*` environment variable.

Hostnames depend on the cluster's networking: the examples below are for public networking. Private endpoints, such as `*.private.confluent.cloud`, are reachable only from networks connected to them, so Terraform must run in one of those networks, or reach them through a proxy in one.

| Destination | Used for | Address from | Authentication |
|---|---|---|---|
| Kafka REST endpoint, for example `pkc-<id>.<region>.<cloud>.confluent.cloud` | Topics, ACLs, cluster settings, cluster links and mirror topics. `confluent_api_key` also uses it to check that a new Kafka API key works, unless `disable_wait_for_ready` is set. | `kafka_rest_endpoint` (`KAFKA_REST_ENDPOINT`), the resource's `rest_endpoint`, or `IMPORT_KAFKA_REST_ENDPOINT`. `confluent_cluster_link` and `confluent_kafka_mirror_topic` use the `rest_endpoint` in their Kafka cluster blocks. `confluent_api_key` reads the cluster's REST endpoint from the Confluent Cloud API. | Kafka API key, or with OAuth, your identity provider's token |
| Schema Registry endpoint, for example `psrc-<id>.<region>.<cloud>.confluent.cloud` | Schemas, subject and registry settings, schema exporters, and encryption keys (KEKs and DEKs). `confluent_schema` also calls it during plan to check schema changes. | `schema_registry_rest_endpoint` (`SCHEMA_REGISTRY_REST_ENDPOINT`), the resource's `rest_endpoint`, or `IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT` | Schema Registry API key, or with OAuth, your identity provider's token |
| Stream Catalog, on the Schema Registry endpoint by default | Tags, business metadata, their bindings, and entity attributes | `catalog_rest_endpoint` (`CATALOG_REST_ENDPOINT`), else the Schema Registry endpoint; the resource's `rest_endpoint`; or `IMPORT_CATALOG_REST_ENDPOINT`, else `IMPORT_SCHEMA_REGISTRY_REST_ENDPOINT` | Schema Registry API key, or with OAuth, your identity provider's token |
| Flink endpoint, for example `flink.<region>.<cloud>.confluent.cloud` | Flink statements, connections and materialized tables | `flink_rest_endpoint` (`FLINK_REST_ENDPOINT`), the resource's `rest_endpoint`, or `IMPORT_FLINK_REST_ENDPOINT` | Flink API key, or with OAuth, your identity provider's token |

## Other Services

| Destination | Used for | When | Authentication |
|---|---|---|---|
| Your identity provider's token URL (`oauth_external_token_url` in the `oauth` block) | Getting an access token with the OAuth client credentials grant | OAuth with a token URL only: each time Terraform configures the provider, and again shortly before the token expires. Not called when you set `oauth_external_access_token`. | Client ID and secret |
| Cloud storage: Amazon S3, Azure Blob Storage or Google Cloud Storage | Uploading the files of `confluent_flink_artifact`, `confluent_connect_artifact`, `confluent_custom_connector_plugin` and `confluent_custom_connector_plugin_version`, to an upload URL that the Confluent Cloud API returns | When one of these resources is created | The upload URL itself (presigned) |

`confluent_tf_importer` also runs the `terraform` binary found on your `PATH`, to run `terraform state replace-provider` on the state file it writes. That command doesn't contact Confluent Cloud, but Terraform may run its own update check (`checkpoint-api.hashicorp.com`) unless `CHECKPOINT_DISABLE` is set.

## Proxies and Certificates

Every connection above uses the proxy set by the standard `HTTPS_PROXY` and `NO_PROXY` environment variables (or their lowercase forms) where Terraform runs. All destinations use HTTPS, so `HTTP_PROXY` applies only to an `http://` URL you configure.

Server certificates are checked against the operating system's trusted certificates; on Linux, `SSL_CERT_FILE` and `SSL_CERT_DIR` can point to others. If your proxy inspects TLS, its CA certificate must be trusted this way.

## Usage Telemetry

The provider sends a usage event for each resource operation to the Confluent Cloud API. An event contains:

* the resource type and the operation (create, read, update, delete or import)
* its start time, duration, and order within the run
* whether it failed and, if the provider crashed, a stack trace of the provider's own source files and line numbers
* the names of changed attributes, never their values
* the provider and Terraform versions, the operating system and architecture, and a random ID for the provider process

Data sources aren't reported.

Events are sent only when all of these are true:

* `CONFLUENT_DISABLE_PROVIDER_ANALYTICS` isn't set. Set it to any non-empty value, for example `true`, to turn telemetry off.
* `endpoint` is `https://api.confluent.cloud`, the default. Any other value turns telemetry off.
* The provider is configured with a Cloud API key and secret (in the provider block or their environment variables), or with OAuth.
* The operation made at least one request to the Confluent Cloud API or a cluster endpoint, and all of those went to `*.confluent.cloud` hosts. Identity provider and cloud storage requests aren't counted.

Telemetry goes to the same host as the API, so turn it off with the environment variable rather than a firewall rule. Sending an event never blocks or fails a Terraform operation: an event that can't be sent is dropped.
