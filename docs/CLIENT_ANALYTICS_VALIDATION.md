# Client Analytics — End-to-End Validation (TFCA-C2)

This is a developer/operator reference, **not** Terraform Registry content. It records
the end-to-end validation of the provider's client-analytics (usage telemetry) pipeline
and the consumption caveats the Grafana dashboards must carry. It backs
[APIE-1574](https://confluentinc.atlassian.net/browse/APIE-1574) (TFCA-C2).

## The pipeline being validated

```
resource CRUD/import
  → telemetry wrapper builds a metadata-only telemetry.Usage
  → publishedTelemetryReporter (opt-out gate)
  → bounded-worker Transport (fixed pool, shallow queue, log-and-drop)
  → SDK poster  → POST https://api.confluent.cloud/terraform-usage/v1/usages
  → cc-cli-service CollectTerraformUsage → OTel span named "<resource_type>.<operation>"
  → heracles-prod-amp → Grafana
```

Reporting is enabled for a process **only** when all of these hold (see
`internal/provider/telemetry_runtime.go`):

1. the preview opt-in `CONFLUENT_PROVIDER_ANALYTICS_PREVIEW` is set (temporary, removed at go-live);
2. the opt-out `CONFLUENT_DISABLE_PROVIDER_ANALYTICS` is unset;
3. the configured `endpoint` is exactly `https://api.confluent.cloud` (any other host, including gov/FedRAMP and staging, disables);
4. a top-level Cloud identity is configured (Cloud API key/secret or an OAuth/STS bearer);
5. the provider is not in acceptance/live test mode (`TF_ACC` / `TF_ACC_PROD`).

Because condition 5 forces telemetry off during `make testacc` and `make live-test*`,
the ordinary acceptance suite cannot be the validation vehicle — hence this distinct pass.

## What is validated in-repo (hermetic, runs in CI)

`internal/provider/telemetry_e2e_validation_test.go` exercises the **enabled** path
without a live backend by publishing an enabled runtime directly. Every event travels
through the real bounded-worker `telemetry.Transport`; the tests run serially and keep
exactly one event in flight, so the shallow, lossy transport queue never drops and
delivery order equals invocation order. This does **not** revert the shipped test-mode
gate — the gate stays intact while the real `New()`-wrapped resources and the real
transport are driven directly.

| Test | Covers | Asserts |
|------|--------|---------|
| `TestTelemetryE2E_AllManagedResourcesEmitCorrectPayloads` | Every managed resource, every wrapped entry point (create/read/update/delete/import), driven over the real transport one event in flight | Exactly one event per entry point, correct `resource_type`/`operation`, one stable process `run_id`, and strictly increasing `sequence` values |
| `TestTelemetryE2E_NamedResourcesLifecycleOverTransport` | A control-plane resource (`confluent_environment`) and a data-plane resource (`confluent_kafka_topic`), each through a full create → read → update → delete → import lifecycle over the same real transport | Each operation delivers exactly one event with the correct `resource_type`/`operation`/`run_id` and a strictly increasing `sequence` |
| `TestTelemetryE2E_ForcedPanicProducesCrashPayload` | A forced panic in a wrapped resource, delivered end to end through the enabled reporter and the real transport | Crash payload with `error: true`, a trimmed stack trace, and the correct resource type/operation |

The SDK-to-wire mapping (the JSON body and the `POST /terraform-usage/v1/usages` path) is
covered separately by `internal/provider/telemetry/client_test.go`.

These tests need no `TF_ACC`; they run in both the unit tier (`make test`) and
`make testacc`. Run just them:

```bash
make testacc TESTARGS="-run TestTelemetryE2E_"
# or directly (no TF_ACC needed), with -race:
go test ./internal/provider -run 'TestTelemetryE2E_' -race -count=1 -v
```

### Result of the validation run

A representative local run (point-in-time; the tests are deterministic, so any run
reproduces the same structure). Reproduce with the commands above and read the `t.Logf`
lines the tests print.

- **Managed resources validated:** **65** — the full `ResourcesMap`.
- **Total events delivered over the real transport (one `go test` process):** **328** —
  **317** from the breadth sweep (one per wrapped entry point), **10** from the two
  named-resource lifecycles, and **1** crash payload. Every event carried the correct
  `resource_type` and `operation`, one stable process `run_id`, and a unique, strictly
  increasing `sequence`.
- **Breadth sweep:** 65 resources / 317 wrapped entry points, one correctly-typed event
  each, sequences `1–317`.
- **Named-resource lifecycles** over the transport (illustrative sequence numbers from one
  ordered run): `confluent_environment` — CREATE 318, READ 319, UPDATE 320, DELETE 321,
  IMPORT 322; `confluent_kafka_topic` — CREATE 323, READ 324, UPDATE 325, DELETE 326,
  IMPORT 327.
- **Forced panic:** a crash payload (`error: true` + a trimmed, path-redacted stack trace)
  delivered end to end.
- **Sample sweep payload:** `resource_type=confluent_<resource> operation=CREATE
  run_id=<uuid> sequence=1 os=darwin arch=arm64 provider_version=<version> error=true` (the
  first resource varies with Go map-iteration order). The hermetic tests drive entry points
  with nil arguments, so the inner CRUD errors or panics and `error` is `true` — expected,
  and it does not affect the envelope fields under test
  (`resource_type`/`operation`/`run_id`/`sequence`).

> **Count note:** the design doc and this ticket say "64 managed resources"; the current
> `ResourcesMap` has **65** (one resource has been added since the design was written).
> The validation covers whatever the map contains, and the count pin in
> `TestWrapResourcesMap_CoversAllManagedResourcesAndExcludesDataSources` is the source of truth.

## Completing the acceptance criterion against a live backend (manual runbook)

C2's literal acceptance — a full plan → apply → import → destroy cycle whose events land
in a real `cc-cli-service` — needs the `terraform-usage/v1` route deployed (Epic A3/A4/A5)
and a real Cloud identity. It cannot run in CI. Perform it manually once the route is live:

1. **Build the provider and point Terraform at it** via a `dev_overrides` block (see
   [DEVELOPING.md](DEVELOPING.md)).
2. **Enable reporting.** Set `CONFLUENT_PROVIDER_ANALYTICS_PREVIEW=1`, leave
   `CONFLUENT_DISABLE_PROVIDER_ANALYTICS` unset, and provide a top-level Cloud API
   key/secret.
3. **Choose the backend:**
   - *Prod public endpoint* (recommended once A5's route is in prod): keep the default
     `endpoint = https://api.confluent.cloud` and use a throwaway/test org. No code change.
   - *Staging `cc-cli-service`*: the endpoint gate deliberately enables reporting only for
     the exact prod host, so a staging host disables telemetry. To validate against
     staging, apply a **local-only** relaxation of the endpoint check in
     `telemetryOptOut` (point it at the staging host) and set `endpoint` to that host.
     **Do not commit this change** — it exists only to exercise a non-prod backend.
4. **Drive the lifecycle** for a control-plane and a data-plane resource, e.g.
   `confluent_environment` and `confluent_kafka_topic`:

   ```bash
   terraform apply         # create + read
   terraform apply         # after editing an updatable attribute → update
   terraform import ...     # import
   terraform destroy       # delete
   ```

5. **Confirm the events landed** with the expected `resource_type`/`operation`/`run_id`/
   `sequence` in `cc-cli-service` logs, in Tempo, or via
   `traces_span_metrics_calls_total{service_name="cc-cli-service"}` in AMP. Remember a
   `plan` and its `apply` are separate processes with **different** run IDs (see caveats).
6. **Crash path (optional):** temporarily inject a `panic(...)` into one resource's CRUD
   locally and confirm a payload with `error: true` and a stack trace arrives (and, if A3
   kept it in scope, that a Jira ticket is auto-filed). Revert the injected panic.

## Consumption caveats — hand-off to Epic D (dashboards)

None of these are fixable provider-side; each is easy to misread. Epic D (TFCA-D3) must
carry them onto the dashboard itself — as panel descriptions or a dashboard-level text
panel — not merely in a ticket.

1. **`run_id` does not join a plan to its apply, and does not join aliased provider
   instances.** Terraform launches a fresh provider subprocess per phase and per alias, so
   each gets its own run ID. Do not build "plan vs apply" or cross-alias correlations on it.
2. **A read-only run is indistinguishable from a no-drift refresh, an import/migration
   workflow, or an idle customer.** The plugin protocol gives the provider no signal for
   which CLI subcommand launched it, so a read-vs-write ratio is not a valid activity or
   adoption proxy on its own.
3. **A truncated apply (an early resource fails, later graph nodes never execute) is
   indistinguishable from a genuinely small apply.** There is no "planned but not attempted"
   marker in the event stream, so event count per run is not a reliable proxy for apply size.
4. **Reporting is deliberately lossy, with a systematic under-reporting bias.** The
   transport uses a fixed worker pool, a shallow queue, drop-on-full, no retries, and there
   is no drain hook before the provider subprocess is killed. Resources late in a large
   apply's dependency graph are therefore structurally more likely to be mid-flight or
   unattempted at exit and are under-reported relative to early ones. A drop in event volume
   is expected behavior, not evidence of an incident — prefer rate-based signals over
   volume-based ones (see TFCA-D4).

## References

- Design doc: [TF Provider: Add Client Analytics to TF Provider](https://confluentinc.atlassian.net/wiki/spaces/AEGI/pages/5964529933/TF+Provider+Add+Client+Analytics+to+TF+Provider)
- Ticket breakdown: [TF Provider Client Analytics (INIT-17732) — Ticket Breakdown](https://confluentinc.atlassian.net/wiki/spaces/AEGI/pages/6041797089/)
- Provider-side implementation: `internal/provider/telemetry_wrapper.go`, `internal/provider/telemetry_runtime.go`, `internal/provider/telemetry/`
