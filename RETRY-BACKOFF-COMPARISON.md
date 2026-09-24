# Retry backoff strategy comparison — INC-13517

Branch: `jpriyatam/connector-get-and-jitter-backoff`. Covers `RetryableClientFactory.CreateRetryableClient`
(`internal/provider/factory_utils.go`), which backs every SDK client's `http.Client`
(`connectV1Cfg`, `iamV2Cfg`, ... — see `provider.go`), not just Connect.

**Assumption used for every number below:** the Confluent Cloud API does not return a `Retry-After`
header on 429/5xx responses. (Row 4 below already uses it for free if that turns out to be wrong
later — see Open Item 2 at the bottom.)

## Current config — what's actually running today, on `master`, unmodified

```
RetryWaitMin = 1s     (defaultRetryWaitMin, hardcoded in go-retryablehttp)
RetryWaitMax = 30s    (defaultRetryWaitMax, hardcoded in go-retryablehttp)
RetryMax     = 4      (provider's "max_retries" — default 4, user-raisable, no ceiling enforced)
Backoff      = DefaultBackoff, applied uniformly to every retryable status (429 and 5xx alike)
```

This is **Option 1** in the table below. It's deterministic — every caller computes the identical
wait for a given attempt number, which is the root cause of INC-13517: 350 `confluent_connector`
resources all hit the same rate limit at once, all computed the same wait, and all retried at the
same instant, re-tripping the limit together, repeatedly.

## Master comparison table

All rows use `min=1s`. "Attempt N" shows the wait at that attempt for `RetryMax=4` (deterministic
value, or `[low–high]` range for a jittered function). "Totals @ RM4"/"@ RM10" are the sum across
a full retry sequence at `RetryMax=4` (today's default) and `RetryMax=10` (illustrating a customer
raising `max_retries` to survive a rate-limit incident — legal today, since the setting has a floor
of 4 but no ceiling). "Relative spread" is the coefficient of variation (`std / mean`) of the wait
at a single attempt — how spread out the random draw is relative to its own size. For rows 2–7 it's
the same number at every attempt (jitter there is a scaled or proportionally-shifted uniform draw,
and CV is scale-invariant), so one value per row is exact, not an approximation. Row 8 is the
exception — its floor is a fixed absolute value (`min`) rather than one that scales with the cap,
so CV grows attempt over attempt instead of staying constant; its cell shows that range. Higher =
more desynchronized = better at breaking up simultaneous retries.

| # | Option | Backoff fn | Scope | `RetryWaitMax` | Attempt 1 | Attempt 2 | Attempt 3 | Attempt 4 | Totals @ RM4 (best / avg / worst) | Totals @ RM10 (avg / worst) | Fixes herd? | Relative spread (std/mean) | True cap at `max`? | Dep. bump? | Status |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| 1 | **Current (baseline)** | `DefaultBackoff` | 429 + 5xx | 30s | 1s | 2s | 4s | 8s | 15s / 15s / 15s | 181s / 181s | ❌ | **0%** *(no randomness)* | ✅ | — | Root cause of the incident |
| 2 | Linear Jitter | `LinearJitterBackoff` | 429 + 5xx | 30s | 1–30s | 2–60s | 3–90s | 4–120s | 10s / ~155s / 300s | ~853s / 1650s | ✅ | 54.0% | ❌ | No | Rejected — unbounded per-attempt growth |
| 3 | Linear Jitter, shrunk max | `LinearJitterBackoff` | 429 + 5xx | 5s | 1–5s | 2–10s | 3–15s | 4–20s | 10s / 30s / 50s | 165s / 275s | ✅ | 38.5% | ❌ | No | Rejected — still not a real cap, still worsens with `RetryMax` |
| 4 | Rate-limit-aware Linear Jitter | `RateLimitLinearJitterBackoff` | 429 + 5xx | 30s | 1–30s | 2–60s | 3–90s | 4–120s | 10s / ~155s / 300s *(= row 2 under this doc's assumption)* | ~853s / 1650s | ✅ | 54.0% *(= row 2)* | ❌ | **Yes (v0.7.8)** | Rejected — same math as row 2 today, plus a version bump, for a `Retry-After` fast path that currently never fires |
| 5 | Full Jitter, everywhere | `fullJitterBackoff` (custom) | 429 + 5xx | 30s | 0–1s | 0–2s | 0–4s | 0–8s | 0s / 7.5s / 15s | 90.5s / 181s | ✅ | **57.7% — highest of any bounded option** | ✅ | No | Considered — correct shape, wider blast radius than the split-scope approach (row 9) |
| 6 | Equal Jitter, everywhere | `equalJitterBackoff` (custom) | 429 + 5xx | 30s | 0.5–1s | 1–2s | 2–4s | 4–8s | 7.5s / 11.25s / 15s | 135.75s / 181s | ✅ | 19.3% — a third of row 5's spread | ✅ | No | Considered — minor variant of row 5, avoids near-zero waits, no strong reason to need that here |
| 7 | Full Jitter, 429 only | `fullJitterBackoff` for 429; `DefaultBackoff` (row 1, unchanged) for 5xx | **429 only** | 30s | 0–1s | 0–2s | 0–4s | 0–8s | 0s / 7.5s / 15s (429 path); 5xx stays exactly row 1: 15s always | 90.5s / 181s (429 path); 5xx stays row 1: 181s always | ✅ | 57.7% (429 path); 0% (5xx path, unchanged) | ✅ | No | *Superseded by row 9* — the `0` end of its range gives a rate limiter no time to recover, wasting a retry attempt (see note below) |
| 8 | Full Jitter, floored at `min`, everywhere | `minFloorFullJitterBackoff` (custom): `wait = min + random(0, capWait − min)` | 429 + 5xx | 30s | **1s (no jitter — degenerate)** | 1–2s | 1–4s | 1–8s | 4s / 9.5s / 15s | 10s / 95.5s / 181s | ⚠️ Partial | 0% → 44.9%, *rising with attempt (see note above)* | ✅ | No | Considered — never spends a retry attempt on a wait too short to possibly succeed, but wider blast radius than row 9 |
| 9 | **Full Jitter, floored at `min`, 429 only** | `minFloorFullJitterBackoff` for 429; `DefaultBackoff` (row 1, unchanged) for 5xx | **429 only** | 30s | 1s (429 path, degenerate) | 1–2s | 1–4s | 1–8s | 4s / 9.5s / 15s (429 path); 5xx stays row 1: 15s always | 10s / 95.5s / 181s (429 path); 5xx stays row 1: 181s always | ✅ | 0% → 44.9% (429 path); 0% (5xx, unchanged) | ✅ | No | **✅ IMPLEMENTED — row 8's floor, scoped like row 7** |

**Rows 2–4 vs. rows 1/5/6/7/8/9, at `RetryMax=10`:** rows 2 and 4 (unbounded linear-jitter math)
grow **quadratically** with `RetryMax` — the multiplier climbs every attempt, so the per-attempt
ceiling itself keeps growing, no plateau. Every other row grows only **linearly** — once the
exponential term exceeds `max`, every extra attempt just adds one more flat, `max`-bounded wait.
That's the concrete reason rows 2–4 were rejected in favor of the capped-jitter shape (rows 5–9).

**Split scope (rows 7/9), not uniform (rows 5/6/8):** 429 specifically means a shared rate limit —
many callers failing for the same reason at the same moment, exactly this incident's shape. 5xx
*can* also be correlated across callers, so this isn't a claim 5xx is immune to the same herd risk
in general — it's a deliberate scoping choice to touch only the status this incident implicates and
leave every other retryable code, and every other part of this client's behavior, byte-for-byte
unchanged from what's shipped today. `DefaultBackoff` itself already groups 429 and 503 for its own
`Retry-After` check, so this isn't a novel split.

**Row 7 → row 9: the `0` end of a jitter range isn't just aggressive, it's likely wasted.** A retry
that lands near 0s after a 429 gives the rate limiter essentially no elapsed time to recover, so
it's very likely to get 429'd again — burning one of only 4 retry attempts (`RetryMax=4`) for no
real chance of success, rather than trading anything meaningful for the extra spread. The
*aggregate* benefit across many simultaneous callers still holds even with that low end included
(other callers in the same instant draw larger values and get through, thinning the herd for the
next round), but for any single caller, its low-end draws are effectively donated to the spread
rather than being productive attempts. This is exactly why row 7 (the first thing implemented) was
superseded by row 9: same split scope, but floored at `min` so no draw is ever spent on a wait too
short to possibly succeed. The cost, visible in row 9's Attempt-1 column, is that the very first
retry after the incident's initial synchronized burst still can't be jittered at all — `capWait`
there already equals `min`, so `[min, capWait]` collapses to a single point.

**What this looks like for 3 connectors hit by the same 429 at once** (one illustrative sample —
actual draws vary every time, that's the point): under row 1 today, all three compute the same
wait and land on the API at the same 4 instants (1s, 2s, 4s, 8s after failure) every single round.
Under row 9 (implemented), attempt 1 is still forced to exactly 1.0s for all three — but attempts
2–4 draw independently within their `[min, capWait]` ranges: e.g. one run might see connector A
retry at 1.0s/1.7s/2.9s/5.2s, B at 1.0s/1.4s/1.8s/7.1s, and C at 1.0s/1.1s/3.6s/1.4s — the first
round stays synchronized, but the three connectors spread across up to 9 different instants over
the remaining three rounds instead of 3 shared ones, with no change to the worst-case ceiling.

## Open items

1. **Is `min=1s` actually long enough for the rate limiter to recover?** The floor chosen for rows
   8/9 (implemented) assumes `RetryWaitMin=1s` is roughly the right order of magnitude for the
   limiter to free up capacity — but `1s` is a generic `go-retryablehttp` default, not a measured
   property of Confluent Cloud's Connect API rate limiter. `INC-13517-NOTES.md` cites a "10 req/s"
   default per-user limit, which could be a hard 1-second window (nothing recovers until the window
   rolls over — `1s` would be about right) or a continuously-refilling token bucket (~1 token every
   100ms — in which case even sub-second waits have *some* chance, and `min=1s` may be more
   conservative than necessary). Not verified in this investigation; worth confirming with the
   Connect team or by inspecting real 429 response headers before trusting this floor value.

2. **`Retry-After` handling.** `minFloorFullJitterBackoff` (rows 8/9, implemented as row 9) computes
   `capWait := DefaultBackoff(...)`, which means if the API ever *does* return a `Retry-After`
   header on 429 (row 1/4's behavior), `capWait` becomes that exact server-provided value — and
   this function would then either jitter under it or clamp it up to `min`, neither of which honors
   the server's exact instruction. Every number in this doc assumes that header is never sent. If
   that assumption turns out to be wrong, this function needs to check for the header itself and
   return it as-is, the way `DefaultBackoff`/row 4 already do. Flagged in the function's own doc
   comment in `factory_utils.go`, not yet fixed.
