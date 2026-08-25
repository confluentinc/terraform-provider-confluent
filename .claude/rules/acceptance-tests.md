---
paths:
  - "internal/provider/*_test.go"
---

# Test Conventions (Acceptance, WireMock, Live)

Tests live alongside the resources in `internal/provider/`. There is no `golangci-lint` in this repo,
so tests are the primary safety net — a schema or CRUD change without test coverage is a review
blocker, not a follow-up.

## Read a neighbor first

Copy the structure of the closest existing `*_test.go` for the resource you're touching. The
WireMock scaffold (stubbing the Confluent Cloud endpoints the resource calls) is fiddly to write from
scratch and consistent across siblings — mirror it rather than inventing one.

## Three tiers

- **Acceptance tests** (`internal/provider/<name>_test.go`): the SDKv2 harness, `resource.Test` or
  `resource.ParallelTest`, gated by `TF_ACC=1`. Run via `make testacc` (sets `TF_ACC=1`, uses
  `gotestsum`, coverage, a long timeout, and `-failfast`).
- **WireMock-backed acceptance tests**: the default and preferred form. They stub the API with
  WireMock (see `wiremockTestUtils.go`) so the test runs in CI **without live credentials**. New
  behavior should land here whenever the API interaction can be faithfully stubbed.
- **Live tests** (`internal/provider/<name>_live_test.go`): build tags `live_test,all`, run against
  real Confluent Cloud with `TF_ACC=1 TF_ACC_PROD=1` via `make live-test*`. Test function names end
  in `Live` (or `DriftDetection`) so the runner's `.*Live$|.*DriftDetection$` filter selects them.
  Add or update a live test when real API behavior matters and WireMock can't meaningfully cover it.

## What a good test asserts

- Assert on the **attributes the change actually affects** with `resource.TestCheckResourceAttr`,
  `TestCheckResourceAttrSet`, and friends — not merely that `apply` returns no error.
- Cover the lifecycle the resource supports: create, then (if updatable) an in-place update step, and
  `ImportState` where import is supported.
- For a `ForceNew` attribute, a test step that changes it should show the resource being recreated.
- For a state upgrader, cover the migration path from the prior `SchemaVersion`.

## Running

```bash
make test           # unit tests via gotestsum (no TF_ACC; -v omitted to keep -json framing intact)
make testacc        # acceptance tests: TF_ACC=1, coverage, -failfast, long timeout
make live-test-*    # live tests against real Confluent Cloud (needs credentials + TF_ACC_PROD=1)
```

Do not add `-v` to the CI test invocation — it breaks the `-json` output framing `gotestsum` relies
on.
