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
	"context"
	"net/http"
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

// telemetryAuthFunc builds the per-request auth decorator from the provider's
// top-level Cloud identity ONLY — the OAuth/STS bearer token or the Cloud API
// key/secret, preferring the bearer token when present (the same order the
// provider uses to authenticate to Confluent Cloud APIs; see the ApiContext
// helpers in utils.go). It deliberately never reads resource-scoped credentials
// (Kafka/Schema Registry/Flink/Tableflow API keys), which are not even passed in:
// analytics is attributed to the org-level identity, not a data-plane key. The
// bearer token is snapshotted here rather than read from the live *STSToken at
// send time, so a mid-run token refresh is not picked up (best-effort) and the
// worker goroutines never race the unsynchronized token mutation in utils.go.
// Returns nil when no top-level identity is configured, which
// publishTelemetryRuntime turns into a disabled runtime (TFCA-B7).
func telemetryAuthFunc(cloudAPIKey, cloudAPISecret string, oauth *OAuthToken, sts *STSToken) func(context.Context) context.Context {
	switch {
	case oauth != nil && sts != nil && sts.AccessToken != "":
		token := sts.AccessToken
		return func(ctx context.Context) context.Context {
			return telemetry.TokenAuthContext(ctx, token)
		}
	case cloudAPIKey != "" && cloudAPISecret != "":
		return func(ctx context.Context) context.Context {
			return telemetry.BasicAuthContext(ctx, cloudAPIKey, cloudAPISecret)
		}
	default:
		return nil
	}
}

// publishTelemetryRuntime computes the reporting decision and publishes the
// runtime the resource wrappers read, once at the end of provider configuration.
//
// Reporting is enabled only when the process is not opted out and is on the
// production endpoint (TFCA-B6), a top-level Cloud identity is configured to
// attribute it (TFCA-B7), and the provider is not running an acceptance or live
// test — live tests use the production endpoint with real credentials, so the
// real transport must stay off there. When enabled the sink is the bounded-worker
// transport; otherwise it is nil and the late-binding reporter drops every event.
func publishTelemetryRuntime(ctx context.Context, endpoint, userAgent, cloudAPIKey, cloudAPISecret string, oauth *OAuthToken, sts *STSToken, testMode bool) {
	// Auth scoping (TFCA-B7): reporting uses only the top-level Cloud identity. If
	// none is configured (for example a provider set up with only resource-scoped
	// Kafka credentials, which are never passed here), authFunc is nil and
	// reporting is a no-op — not an error.
	authFunc := telemetryAuthFunc(cloudAPIKey, cloudAPISecret, oauth, sts)
	disabled := telemetryOptOut(endpoint) || authFunc == nil || testMode
	rt := &telemetryRuntime{config: telemetry.NewConfig(disabled)}
	if !disabled {
		poster := telemetry.NewSDKPoster(endpoint, &http.Client{}, userAgent, authFunc)
		rt.reporter = telemetry.NewTransport(poster, ctx)
	}
	publishedTelemetry.Store(rt)
}
