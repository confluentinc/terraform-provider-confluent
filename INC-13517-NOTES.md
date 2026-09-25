# INC-13517 — `confluent_connector` refresh fix

## Situation

MasterControl Labs (Confluent Cloud org 35760) hit sustained HTTP 429s from the Connect v1 API
mid-way through a monthly production Terraform release, blocking their deploy. Their Terraform
state has ~350 `confluent_connector` resources on one cluster (`lkc-12kv6j`).

Root cause, confirmed against the customer's own Terraform log and against this provider's source
(`internal/provider/resource_connector.go`, as of v2.86.6):

- `confluent_connector`'s `Read()` does not fetch a single connector. It calls
  `ListConnectv1ConnectorsWithExpansions` (`GET .../connectors?expand=info,status,id` — the full,
  unpaginated list of every connector in the cluster) and filters the result client-side for the
  one connector it wants.
- This runs **once per `confluent_connector` resource on every refresh**, so a state with N
  resources issues N full-list calls, each re-fetching and re-marshaling all N connectors. That's
  N² request/response volume, growing quadratically with fleet size — not the "constant work,
  just more of it" the customer expected from "we only added connectors."
- Their connector count grew past the point where this pattern's request rate exceeds the
  platform's default per-user rate limit (10 req/s), which is why the same Terraform code that
  "worked for years" started failing this month with no code change on either side.
- The customer's raw log confirms this directly: 358 distinct `confluent_connector.this[...]`
  resources all failed against the exact same list URL, never the per-connector endpoint.

## Fix

`readConnectorAndSetAttributes` (`internal/provider/resource_connector.go`) now branches on
`d.IsNewResource()`:

- **Create / Import** (`IsNewResource() == true`): unchanged — still uses the full list+expand
  call, because the connector's ID genuinely isn't known yet and has to be discovered there.
- **Everything else** — i.e. every routine refresh of a connector already in state, which is the
  actual N² hot path — now calls two by-name endpoints instead:
  - `GetConnectv1ConnectorConfig(name)` → `GET .../connectors/{name}/config`
  - `ReadConnectv1ConnectorStatus(name)` → `GET .../connectors/{name}/status`

  Both already exist in the SDK and are already used elsewhere in this exact provider (the
  by-name status call backs the create/pause/resume wait loops in `utils_wait.go`; the by-name
  config PUT backs `connectorUpdate`), so this isn't new API surface — just wiring `Read()` to use
  what already exists instead of the list endpoint.

This turns N full-fleet list calls per refresh into N cheap by-name calls, and each response goes
from "every connector's full config + status" down to just the one connector's.

Added `executeConnectorReadByName` and a shared `handleConnectorReadError` helper (both new,
small, in the same file) to keep the not-found/removed-from-state handling identical between the
two paths.

### Immediate customer-side mitigation (no provider change needed)

`terraform apply -refresh=false` removes the list calls entirely for this run, since refresh is
the only thing that triggers them and their release process already trusts the plan output.

### Test coverage

Updated `resource_connector_managed_test.go`'s WireMock scenario with new stubs for the by-name
config/status endpoints at every scenario state where a routine (non-new-resource) refresh can now
land: post-create, post-config-update, post-offsets-update. Added two new fixtures:
`read_connector_config.json`, `read_updated_connector_config.json`.

**Not yet verified end-to-end** — this environment has no Docker, so `TestAccManagedConnector`
(the WireMock-backed acceptance test) could not actually be run. The stub sequencing was reasoned
through by hand against the SDK's `IsNewResource()`/`MarkNewResource()` semantics, but should be
confirmed with `make testacc` before this is relied on or merged.
