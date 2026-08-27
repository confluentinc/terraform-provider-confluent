# Terraform Provider for Confluent

## Project Overview

The Confluent Terraform provider manages Confluent Cloud resources (Kafka clusters, topics, API
keys, Flink, Schema Registry, networking, RBAC, and more) through Terraform. It is published to the
Terraform Registry and used by customers to provision real infrastructure, so schema correctness and
backward compatibility are hard constraints.

These instructions exist to keep GitHub Copilot PR reviews focused on the invariants that matter in
this codebase. Author-facing setup lives in `docs/DEVELOPING.md`; the review-relevant conventions
are captured here and in `.claude/rules/`.

## Architecture (what reviewers need to know)

- **Built on terraform-plugin-sdk/v2** (`helper/schema`), _not_ the newer terraform-plugin-framework.
  Resources are `*schema.Resource` values with a `Schema map[string]*schema.Schema` and CRUD
  functions (`CreateContext`/`ReadContext`/`UpdateContext`/`DeleteContext`) returning
  `diag.Diagnostics`.
- **Everything lives in one flat package, `internal/provider/`**: ~240 `resource_*.go`, ~160
  `data_source_*.go`, plus `constants.go` (shared strings, including deprecation messages like
  `deprecationMessageMajorRelease3`), `utils.go`, `factory*.go`, `state_upgraders*.go`, and
  `wiremockTestUtils.go`.
- **Registration is in `internal/provider/provider.go`**: `New(version, userAgent string)` builds
  the `*schema.Provider`, whose `ResourcesMap` and `DataSourcesMap` register every resource and data
  source by its Terraform type name (e.g. `confluent_kafka_cluster`). A resource that exists in a
  file but isn't in the map does not exist to Terraform.
- **Three test tiers**: unit + acceptance tests (`internal/provider/*_test.go`, gated by `TF_ACC=1`,
  most backed by **WireMock** so they run without live credentials); and live tests
  (`*_live_test.go`, build tags `live_test,all`, run against real Confluent Cloud).
- **Docs are hand-maintained.** `tfplugindocs` is not wired into CI, so `docs/` and `examples/` do
  not regenerate automatically and drift silently. Doc/example accuracy is a review responsibility.

## Code Review Guidelines (GitHub PR Reviews)

When reviewing pull requests, focus on the checkpoints below. Formatting is owned by
`make checkfmt` (goimports) — don't comment on import ordering or gofmt-able whitespace. Note there
is **no `golangci-lint`** in this repo, so logic-level review carries more weight than it would in a
repo with a rich linter.

### 1. Resource and data-source registration

A new resource or data source requires **both** a `*schema.Resource` constructor and an entry in the
matching map in `provider.go`:

- [ ] New resource → entry in `ResourcesMap` keyed by its `confluent_*` type name
- [ ] New data source → entry in `DataSourcesMap`
- [ ] The type name matches the file name and the docs page (`confluent_kafka_cluster` ↔
      `resource_kafka_cluster.go` ↔ `docs/resources/kafka_cluster.md`)

A constructor with no map entry is dead code; a map entry with a typo'd type name breaks the
resource for every user.

### 2. Schema attribute correctness

The `Schema` map is the contract with users and with Terraform's plan/apply engine. Review each new
or changed attribute for:

- [ ] Correct `Required` / `Optional` / `Computed` combination. Common mistakes: `Required` +
      `Computed` together (invalid), an input the API assigns marked `Required` instead of
      `Optional`+`Computed`, or a server-set value missing `Computed`.
- [ ] `ForceNew: true` on any attribute that cannot be updated in place — omitting it makes Terraform
      attempt an impossible in-place update; adding it spuriously forces destructive recreation.
- [ ] `Sensitive: true` on secrets (API secrets, passwords, tokens) so they're redacted in plan
      output.
- [ ] Validation (`ValidateFunc` / `ValidateDiagFunc`) where the API constrains values.
- [ ] `DiffSuppressFunc` where the server canonicalizes a value (case-folding a region/enum via
      `strings.EqualFold`, normalizing a query). Missing it turns a benign server rewrite into
      permanent drift; adding it spuriously hides a real user change.
- [ ] Nested blocks use the established `*schema.Resource` `Elem` pattern consistent with sibling
      resources. A `TypeList`/`TypeSet` missing its `Elem`, or a dropped `Optional` on an element,
      changes the set hash.

### 3. Deprecation and backward compatibility

Customers pin the provider and run it against live infrastructure. Treat these as blockers unless
they're explicitly gated on a major release:

- [ ] No removed or renamed attribute/resource without a deprecation cycle. Deprecate via
      `Deprecated:` / `DeprecationMessage:` using the shared strings in `constants.go` (e.g.
      `deprecationMessageMajorRelease3`) rather than inventing a new message.
- [ ] No change that would force unexpected recreation of existing resources (a new `ForceNew` on an
      existing attribute, a changed default).
- [ ] Renaming or restructuring stored state requires a `SchemaVersion` bump plus a state upgrader
      (see checkpoint 4).

### 4. State upgraders

- [ ] Any change to an existing resource's stored shape (renamed attribute, changed nesting, type
      change) bumps `SchemaVersion` and adds a `StateUpgrader` (see `state_upgraders*.go`) so
      existing state migrates cleanly. A shape change with no upgrader corrupts existing users' state
      on the next apply.

### 5. Testing

- [ ] New behavior has an acceptance test (`resource.Test` / `resource.ParallelTest`, `TF_ACC=1`).
      Prefer a WireMock-backed test so it runs in CI without live credentials — mirror the setup in a
      sibling `*_test.go` and `wiremockTestUtils.go`.
- [ ] A live test (`*_live_test.go`, build tags `live_test,all`) is added or updated when the change
      touches real API behavior that WireMock can't meaningfully cover.
- [ ] Tests assert on the attributes the PR changed (via `resource.TestCheckResourceAttr` and
      friends), not just that apply succeeds.

### 6. Documentation and examples

Because docs don't regenerate:

- [ ] New/changed attributes are reflected in `docs/resources/*.md` or `docs/data-sources/*.md`.
- [ ] `examples/` (and `community-contributed-examples/` if touched) still validate against the
      current schema.
- [ ] `CHANGELOG.md` has an entry for user-visible changes.

### 7. Generated / regenerated resources (`cli-terraform-generator`)

Much of this repo's churn is replacing hand-written resources with generated code. A PR is a
generator PR when it touches `.tf-generation-checksum`, carries the `cli-terraform-generator` label,
or a changed `*.go` file starts with `// Code generated by cli-terraform-generator ...; DO NOT EDIT.`.
These PRs almost always claim behavior is unchanged — the review's job is to verify that:

- [ ] **Verify behavior-preservation against the base, don't trust the claim.** The regressions that
      have slipped through: `GetOkExists` vs `GetOk` (the former sends an explicit zero), a `TypeList`
      missing its `Elem`, a dropped `Optional: true` on an element (changes a `TypeSet` hash).
- [ ] **A `ValidateFunc` or description that looks wrong is usually spec-faithful.** The generator
      derives them from the OpenAPI minispec; `StringLenBetween(0, N)` instead of `StringIsNotEmpty`
      means the spec omits `minLength`. The fix is a spec change + regenerate, **not** a hand-edit —
      a generated file says `DO NOT EDIT` and a hand-patch is lost on the next run.
- [ ] **Fixes route upstream.** Template bugs → `cli-terraform-generator`; validation/descriptions →
      the minispec. A "fixed in the generator" claim should link the upstream PR.

### 8. Error handling and async status

No linter guards these, and they're where the highest-value bugs hide:

- [ ] Errors aren't swallowed — a non-404 in `Read` must not be treated as "resource gone" (that
      silently drops it from state). 404/403 detection uses the `utils.go` helpers, not a proxy like
      schema presence.
- [ ] Response bodies are handled correctly (close the original `ReadCloser` before overwriting
      `response.Body`; don't assume a body reads twice).
- [ ] Wait/poll loops (`utils_wait.go`) handle terminal states (`ERROR`, `DISCONNECTED`, `FAILED`)
      explicitly, and don't gate attribute-setting on `Status != nil` in a way that leaves a resource
      in perpetual diff while still `PROVISIONING`.

## Files to skip in reviews (generated or infrastructure)

- `docs/**` when it _is_ being regenerated wholesale (review the schema source instead) — but do
  flag hand-authored doc drift per checkpoint 6.
- `.semaphore/**`, `.goreleaser*.yml`, `.golicense.hcl` — release/CI infrastructure.
- `vendor/**` if present.

## Style preferences (avoid nitpicking)

- `make checkfmt` (goimports) owns formatting. Don't comment on import order or whitespace.
- Don't flag that a change "might fail the build" — Semaphore runs the build and tests; the author
  fixes failures before merge.
- Don't nitpick comment _style_ — but a comment or log/error message the diff has made **stale,
  wrong, or non-inclusive is in scope** (e.g. an "Importing Cloud API Key" log that now also handles
  Global/Tableflow keys). Prefer "update it" over "delete it".
- Focus reviews on schema correctness, state safety, backward compatibility, behavior-preservation on
  regenerations, error/status handling, and test coverage.

## Review checklist

Before approving, confirm:

- [ ] New resource/data source is registered in `ResourcesMap` / `DataSourcesMap` with a type name
      matching its file and docs page.
- [ ] Schema attributes have correct `Required`/`Optional`/`Computed`, `ForceNew` where updates are
      impossible, `Sensitive` on secrets, and `DiffSuppressFunc` where the server canonicalizes.
- [ ] No un-deprecated removal/rename; deprecations reuse `constants.go` messages; no unexpected
      forced recreation of existing resources.
- [ ] Stored-shape changes bump `SchemaVersion` and ship a state upgrader.
- [ ] For a generator PR, behavior-preservation is verified against the base branch, and any
      validation/description fix is routed upstream (spec or generator), not hand-edited.
- [ ] Errors aren't swallowed, response bodies are handled, and wait loops cover terminal states.
- [ ] New behavior has acceptance-test coverage (WireMock-backed where possible), and live tests
      where real API behavior matters.
- [ ] Docs, examples, and `CHANGELOG.md` updated for user-visible changes.
- [ ] PR description claims match the diff — spot-check "behavior identical" / "no schema change"
      assertions — and the `.github/pull_request_template.md` checklist items are filled in (release
      notes, blast radius, live testing, breaking-change and feature-flag status).
