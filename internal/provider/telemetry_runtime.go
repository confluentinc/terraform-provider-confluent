// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"context"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// This file wires client analytics into provider configuration and carries the
// opt-out decision (TFCA-B6).
//
// The wrapper (TFCA-B3) captures its reporter once, at New(), before any
// credentials or endpoint are known. The reporter it captures is the late-binding
// publishedTelemetryReporter below: it reads a process-scoped runtime that
// provider configuration publishes exactly once. This is safe publication, not
// mutual exclusion — the runtime is written before the concurrent CRUD/import
// goroutines start and only read thereafter, and the atomic.Pointer provides the
// happens-before edge. Aliased provider instances run in separate processes and
// share none of this.

const (
	// disableProviderAnalyticsEnvVar opts a process out of client analytics when
	// set to any non-empty value. Read once at configuration. (Final public name
	// is coordinated with the CLI opt-out announcement, TFCA-E2/B9.)
	disableProviderAnalyticsEnvVar = "CONFLUENT_DISABLE_PROVIDER_ANALYTICS"

	// defaultCloudEndpoint is the public Confluent Cloud API origin. Any other
	// endpoint (gov/FedRAMP) default-disables analytics, and that is not
	// overridable in v1 — the non-default population is exactly who the gate
	// protects.
	defaultCloudEndpoint = "https://api.confluent.cloud"
)

// telemetryRuntime is the process-scoped analytics state published once during
// provider configuration.
type telemetryRuntime struct {
	config   telemetry.Config
	reporter telemetryReporter // nil when disabled
}

// publishedTelemetry holds the one runtime for this process. atomic.Pointer
// gives the CRUD goroutines a race-free view of the write that configuration made
// before they started.
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
	// at a mock while leaving the top-level endpoint unset, and those must not emit
	// telemetry to production.
	if endpoint != defaultCloudEndpoint {
		return true
	}
	return false
}

// telemetryAuthFunc builds the per-request auth decorator from the provider's
// top-level Cloud identity ONLY — the Cloud API key/secret or the OAuth/STS
// bearer token — preferring a bearer token when present. It deliberately never
// reads resource-scoped credentials (Kafka/Schema Registry/Flink/Tableflow API
// keys): analytics is attributed to the org-level identity, not a data-plane
// key. It returns nil when no top-level identity is configured, which
// publishTelemetryRuntime treats as a disabled runtime (TFCA-B7).
func telemetryAuthFunc(cloudAPIKey, cloudAPISecret string, oauth *OAuthToken, sts *STSToken) func(context.Context) context.Context {
	switch {
	case sts != nil && sts.AccessToken != "":
		return telemetry.TokenAuthContext(sts.AccessToken)
	case oauth != nil && oauth.AccessToken != "":
		return telemetry.TokenAuthContext(oauth.AccessToken)
	case cloudAPIKey != "":
		return telemetry.BasicAuthContext(cloudAPIKey, cloudAPISecret)
	default:
		return nil
	}
}

// publishTelemetryRuntime computes the opt-out decision, builds the transport
// when enabled, and publishes the runtime for the CRUD goroutines. Called once,
// at the end of provider configuration.
func publishTelemetryRuntime(ctx context.Context, endpoint, userAgent, cloudAPIKey, cloudAPISecret string, oauth *OAuthToken, sts *STSToken, testMode bool) {
	// Auth scoping (TFCA-B7): reporting uses only the top-level Cloud identity. If
	// none is configured (for example a provider set up with only resource-scoped
	// Kafka credentials), authFunc is nil and reporting is a no-op — not an error.
	authFunc := telemetryAuthFunc(cloudAPIKey, cloudAPISecret, oauth, sts)
	// Test-mode gating (TFCA-B8): never report during acceptance or live-production
	// test runs, so an unexpected outbound call can't make those suites slow or
	// flaky (cf. hashicorp/terraform-plugin-sdk#640).
	disabled := testMode || telemetryOptOut(endpoint) || authFunc == nil
	rt := &telemetryRuntime{config: telemetry.NewConfig(disabled)}
	if !disabled {
		poster := telemetry.NewSDKPoster(endpoint, &http.Client{}, userAgent, authFunc)
		rt.reporter = telemetry.NewTransport(poster, ctx)
	}
	publishedTelemetry.Store(rt)
}
