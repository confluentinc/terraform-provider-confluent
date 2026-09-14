// Copyright 2026 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"os"
	"sync/atomic"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// This file carries the client-analytics opt-out decision and publishes it to
// the resource wrappers (TFCA-B6).
//
// The wrapper (TFCA-B3) captures its reporter once, at New(), before any
// credentials or endpoint are known. The reporter it captures is the
// late-binding publishedTelemetryReporter below: it reads a process-scoped
// runtime that provider configuration publishes exactly once. This is safe
// publication, not mutual exclusion — the runtime is written before the
// concurrent CRUD/import goroutines start and only read thereafter, and the
// atomic.Pointer provides the happens-before edge. Aliased provider instances
// run in separate OS processes and share none of this.
//
// The terminal sink behind the gate is the no-op reporter. The network
// transport (TFCA-B5) and its auth scoping (TFCA-B7) land downstream and the
// enabled path forwards to them once wired; until then this file is
// behaviorally transparent — no Usage leaves the process and configuration
// performs no new network or filesystem I/O.

const (
	// disableProviderAnalyticsEnvVar opts a process out of client analytics when
	// set to any non-empty value. Read once at configuration. (Final public name
	// is coordinated with the CLI opt-out announcement, TFCA-B9/E2.)
	disableProviderAnalyticsEnvVar = "CONFLUENT_DISABLE_PROVIDER_ANALYTICS"

	// defaultCloudEndpoint is the public Confluent Cloud API origin. It must match
	// the schema default of the provider's "endpoint" argument (the "endpoint"
	// field in provider.go); the comparison is exact, so any other endpoint
	// (gov/FedRAMP) default-disables analytics, and that is not overridable in
	// v1 — the non-default population is exactly who the gate protects.
	defaultCloudEndpoint = "https://api.confluent.cloud"
)

// telemetryRuntime is the process-scoped analytics state published once during
// provider configuration.
type telemetryRuntime struct {
	// config is the immutable opt-out + run-ID snapshot for this process (the
	// configure-time struct TFCA-B1 defines). Report reads config.Disabled; the
	// RunID is published here for the network transport to consume downstream.
	config telemetry.Config
	// reporter is the sink for an enabled runtime; nil when disabled.
	reporter telemetryReporter
}

// publishedTelemetry holds the one runtime for this process. atomic.Pointer
// gives the CRUD/import goroutines a race-free view of the write configuration
// made before they started.
var publishedTelemetry atomic.Pointer[telemetryRuntime]

// publishedTelemetryReporter is the stable reporter the wrapper captures at
// New(). It forwards to whatever configuration published, and drops when nothing
// is published yet or reporting is disabled.
type publishedTelemetryReporter struct{}

func (publishedTelemetryReporter) Report(u telemetry.Usage) {
	rt := publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled || rt.reporter == nil {
		return
	}
	rt.reporter.Report(u)
}

// telemetryOptOut reports whether analytics reporting is disabled for this
// process. Read once at configuration.
func telemetryOptOut(endpoint string) bool {
	if os.Getenv(disableProviderAnalyticsEnvVar) != "" {
		return true
	}
	// Report only when talking to the real production Confluent Cloud endpoint.
	// Any other endpoint disables reporting: gov/FedRAMP hosts (the population the
	// gate protects, not overridable in v1) and — importantly — an empty endpoint.
	// A real provider always resolves the schema default (defaultCloudEndpoint);
	// an empty endpoint only occurs in tests that point resource-level REST calls
	// at a mock while leaving the top-level endpoint unset, and those must not
	// emit telemetry to production.
	return endpoint != defaultCloudEndpoint
}

// publishTelemetryRuntime computes the opt-out decision and publishes the
// process-scoped runtime for the resource wrappers to read. Called once, at the
// end of provider configuration, before the concurrent CRUD/import goroutines
// run.
//
// When enabled, the runtime forwards to the network transport (TFCA-B5),
// authenticated with the top-level Cloud identity (TFCA-B7). Both land
// downstream; until then the enabled sink is the no-op reporter, so publishing
// an enabled runtime today adds no outbound calls — the opt-out gate is in place
// ahead of the transport it will guard.
//
// Two preconditions must hold before a real transport replaces that no-op sink:
//   - It must be safe for concurrent use: the CRUD/import goroutines call the
//     reporter in parallel under Terraform's default parallelism.
//   - Test-mode suppression (TFCA-B8) must gate it. This endpoint check disables
//     acceptance tests (mock/empty endpoint) but NOT live tests, which run
//     against this production endpoint — so the transport must not be wired in
//     before B8 excludes the test suites.
func publishTelemetryRuntime(endpoint string) {
	disabled := telemetryOptOut(endpoint)
	rt := &telemetryRuntime{config: telemetry.NewConfig(disabled)}
	if !disabled {
		rt.reporter = noopTelemetryReporter{}
	}
	publishedTelemetry.Store(rt)
}
