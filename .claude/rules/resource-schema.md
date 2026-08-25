---
paths:
  - "internal/provider/resource_*.go"
  - "internal/provider/data_source_*.go"
---

# Resource & Data-Source Schema Conventions

Built on **terraform-plugin-sdk/v2** (`helper/schema`), not the plugin framework. Every resource is
a `*schema.Resource` with a `Schema map[string]*schema.Schema` and context-based CRUD functions
returning `diag.Diagnostics`. Data sources are the read-only counterpart.

## Before you start: read a neighbor

There are ~240 resources and ~160 data sources in `internal/provider/`, all in one flat package.
**Before adding or changing one, read the closest existing sibling end to end** — its schema map,
CRUD functions, import support, and its `*_test.go` (usually WireMock-backed). Mimicking the nearest
working sibling is the fastest way to get the shape right; the rules below describe the contract, the
siblings show the lived form. If a sibling contradicts a rule here, prefer the sibling and flag the
rule as drift.

## Registration (required, easy to forget)

A resource/data source only exists once it's in the map in `provider.go`:

- New resource → add to `ResourcesMap`, keyed by its Terraform type name (`confluent_<thing>`).
- New data source → add to `DataSourcesMap`.
- Keep the type name consistent across three places: the map key, the file name
  (`resource_kafka_cluster.go`), and the docs page (`docs/resources/kafka_cluster.md`).

A constructor with no map entry is dead code; a typo'd key breaks the resource address for users.

## Attribute correctness

The schema map is the user contract _and_ what Terraform's plan/apply engine reasons about. For each
attribute:

- **`Required` / `Optional` / `Computed`** must match API semantics:
  - User-supplied and mandatory → `Required`.
  - User-supplied and optional → `Optional` (add `Default` if there's a stable default).
  - Server-assigned → `Computed`.
  - Optional input the server may fill in → `Optional` + `Computed`.
  - Never `Required` + `Computed` together (invalid).
- **`ForceNew: true`** on any attribute the API cannot change in place. Omitting it makes Terraform
  attempt an impossible in-place update; adding it where an update _is_ possible forces needless
  destructive recreation. This is the single highest-impact attribute to get right.
- **`Sensitive: true`** on secrets (API secrets, passwords, tokens) so they're redacted in plan
  output.
- **Validation** via `ValidateFunc` / `ValidateDiagFunc` where the API constrains values.
- **Nested blocks** use the `Elem: &schema.Resource{...}` (or `&schema.Schema{...}`) pattern already
  used by siblings; match `MaxItems`/`MinItems` conventions.

## Deprecation

- Deprecate before removing. Set `Deprecated:` (attribute) or `DeprecationMessage:` (resource) using
  a shared string from `constants.go` (e.g. `deprecationMessageMajorRelease3`) — do not hand-write a
  new deprecation message when a shared one fits.
- Removing or renaming an attribute/resource outright is a breaking change; it belongs in a major
  release, not a minor/patch.

## State versioning

- Any change to an existing resource's **stored shape** (renamed attribute, changed nesting, type
  change) requires bumping `SchemaVersion` and adding a `StateUpgrader` (see `state_upgraders*.go`).
- A shape change with no upgrader corrupts existing users' state on their next `terraform apply` —
  this is a correctness bug, not a nicety.

## Docs don't regenerate

`tfplugindocs` is not wired into CI, so `docs/` and `examples/` do not update automatically. When you
change a schema attribute, update the matching `docs/**` page and any affected `examples/` in the
same change, and add a `CHANGELOG.md` entry for user-visible changes. The `docs-drift` skill compares
docs/examples against the current schema maps.
