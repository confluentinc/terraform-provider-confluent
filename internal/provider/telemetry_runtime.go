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

// Client analytics opt-out: during provider configuration, decide whether usage
// reporting is enabled for this process and publish that decision for the
// resource wrappers to read.

const (
	// disableProviderAnalyticsEnvVar opts the process out of analytics when set to
	// any non-empty value.
	disableProviderAnalyticsEnvVar = "CONFLUENT_DISABLE_PROVIDER_ANALYTICS"

	// defaultCloudEndpoint is the public Confluent Cloud API origin. It must match
	// the schema default of the provider's "endpoint" argument; reporting is
	// enabled only for an exact match, and any other endpoint disables it.
	defaultCloudEndpoint = "https://api.confluent.cloud"
)

// telemetryRuntime is the analytics state published once per process during
// provider configuration.
type telemetryRuntime struct {
	// config holds the run ID and opt-out flag, snapshotted at configuration.
	config telemetry.Config
	// reporter is the sink for an enabled runtime; nil when disabled.
	reporter telemetryReporter
}

// publishedTelemetry holds this process's runtime. The atomic pointer lets the
// concurrent resource operations read it without locking.
var publishedTelemetry atomic.Pointer[telemetryRuntime]

// publishedTelemetryReporter is the reporter the wrapper holds. It forwards to
// whatever configuration published, and drops when nothing is published yet or
// reporting is disabled.
type publishedTelemetryReporter struct{}

func (publishedTelemetryReporter) Report(u telemetry.Usage) {
	rt := publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled || rt.reporter == nil {
		return
	}
	rt.reporter.Report(u)
}

// telemetryOptOut reports whether analytics is disabled for this process.
func telemetryOptOut(endpoint string) bool {
	if os.Getenv(disableProviderAnalyticsEnvVar) != "" {
		return true
	}
	// Enable only for the production endpoint; every other value, including an
	// empty endpoint (seen only in tests) and gov/FedRAMP hosts, disables.
	return endpoint != defaultCloudEndpoint
}

// publishTelemetryRuntime computes the opt-out decision and publishes the runtime
// the resource wrappers read, once at the end of provider configuration.
//
// The enabled sink is a no-op today; a real reporter must be concurrency-safe and
// stay off during test runs (live tests use the production endpoint).
func publishTelemetryRuntime(endpoint string) {
	disabled := telemetryOptOut(endpoint)
	rt := &telemetryRuntime{config: telemetry.NewConfig(disabled)}
	if !disabled {
		rt.reporter = noopTelemetryReporter{}
	}
	publishedTelemetry.Store(rt)
}
