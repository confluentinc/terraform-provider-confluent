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

	// previewProviderAnalyticsEnvVar temporarily gates reporting behind an opt-in:
	// reporting stays off for everyone unless this is set to any non-empty value.
	// Remove it once analytics is enabled by default.
	previewProviderAnalyticsEnvVar = "CONFLUENT_PROVIDER_ANALYTICS_PREVIEW"

	// defaultCloudEndpoint is the public Confluent Cloud API origin. It must match
	// the schema default of the provider's "endpoint" argument, and is one of the
	// endpoints for which reporting is enabled (see telemetryEnabledEndpoints).
	defaultCloudEndpoint = "https://api.confluent.cloud"

	// stagingCloudEndpoint and develCloudEndpoint are the Confluent-operated
	// non-production control-plane origins, matching the SDK's server list. They
	// are enabled alongside production so the analytics pipeline can be validated
	// against staging and development.
	stagingCloudEndpoint = "https://api.stag.cpdev.cloud"
	develCloudEndpoint   = "https://api.devel.cpdev.cloud"
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
// concurrent resource operations read it without locking. A process global is safe
// because Terraform runs each provider configuration (including each alias) in its
// own plugin subprocess, so one process serves exactly one configuration.
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

// telemetryEnabledEndpoints is the set of Confluent-operated control-plane
// origins for which usage reporting is enabled: production, staging, and
// development, mirroring the SDK's server list. Gov/FedRAMP hosts are
// deliberately excluded so telemetry never egresses a FedRAMP boundary.
var telemetryEnabledEndpoints = map[string]bool{
	defaultCloudEndpoint: true,
	stagingCloudEndpoint: true,
	develCloudEndpoint:   true,
}

// telemetryOptOut reports whether analytics is disabled for this process.
func telemetryOptOut(endpoint string) bool {
	if os.Getenv(disableProviderAnalyticsEnvVar) != "" {
		return true
	}
	// Enable only for the Confluent-operated control-plane endpoints; every other
	// value, including an empty endpoint (seen only in tests) and gov/FedRAMP
	// hosts, disables.
	return !telemetryEnabledEndpoints[endpoint]
}

// telemetryAuthFunc builds the per-request auth decorator from the provider's
// top-level Cloud identity only — the OAuth/STS bearer token or the Cloud API
// key/secret, never resource-scoped credentials. It returns nil when no top-level
// identity is configured, which disables reporting. The bearer token is captured
// by value so the background workers never race the provider's token refresh.
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

// publishTelemetryRuntime decides whether reporting is enabled and publishes the
// runtime the resource wrappers read, once at the end of provider configuration.
// Reporting is enabled only when the preview opt-in is set, the process is not
// opted out and is on an enabled Confluent endpoint (production, staging, or
// development), a top-level Cloud identity is configured, and the provider is not
// running a test. When enabled the sink is the
// bounded-worker transport; otherwise it is nil and every event is dropped.
func publishTelemetryRuntime(ctx context.Context, endpoint, userAgent, cloudAPIKey, cloudAPISecret string, oauth *OAuthToken, sts *STSToken, testMode bool) {
	authFunc := telemetryAuthFunc(cloudAPIKey, cloudAPISecret, oauth, sts)
	// Temporary opt-in gate: keep reporting off until the preview flag is set.
	previewOptIn := os.Getenv(previewProviderAnalyticsEnvVar) != ""
	disabled := !previewOptIn || telemetryOptOut(endpoint) || authFunc == nil || testMode
	rt := &telemetryRuntime{config: telemetry.NewConfig(disabled)}
	if !disabled {
		poster := telemetry.NewSDKPoster(endpoint, &http.Client{}, userAgent, authFunc)
		rt.reporter = telemetry.NewTransport(poster, ctx)
	}
	publishedTelemetry.Store(rt)
}
