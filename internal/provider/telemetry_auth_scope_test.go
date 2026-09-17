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
	"testing"

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// TestTelemetryDisabledWithoutTopLevelIdentity checks that a provider with no
// top-level Cloud identity (e.g. only resource-scoped Kafka credentials) reports
// nothing: the runtime is disabled with no transport, and reporting is a no-op.
func TestTelemetryDisabledWithoutTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")
	t.Setenv(previewProviderAnalyticsEnvVar, "1")

	// Everything else opens the gate, so the missing identity is the only thing
	// disabling here (no Cloud key and no OAuth/STS token).
	publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "", "", nil, nil, false)

	rt := publishedTelemetry.Load()
	if rt == nil {
		t.Fatal("expected a published runtime")
	}
	if !rt.config.Disabled {
		t.Errorf("runtime must be disabled when no top-level Cloud identity is configured")
	}
	if rt.reporter != nil {
		t.Errorf("no transport should be built without a top-level identity, got %T", rt.reporter)
	}

	// With no transport the reporter is a no-op: it must not panic.
	publishedTelemetryReporter{}.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic", Operation: telemetry.OperationCreate})
}

// TestTelemetryEnabledWithTopLevelIdentity is the positive control: the same
// inputs plus a Cloud API key yield an enabled runtime with a live transport.
func TestTelemetryEnabledWithTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")
	t.Setenv(previewProviderAnalyticsEnvVar, "1")

	publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
	rt := publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled {
		t.Fatalf("expected an enabled runtime with a top-level identity, got %+v", rt)
	}
	// The enabled path must build the real transport, not a no-op reporter (both
	// are non-nil), so assert the concrete type.
	tr, ok := rt.reporter.(*telemetry.Transport)
	if !ok {
		t.Fatalf("enabled runtime must use the real transport, got %T", rt.reporter)
	}
	// Stop the transport's worker goroutines started for this test.
	tr.Close()
}

// TestTelemetryAuthFuncScoping checks telemetryAuthFunc selects the right scheme
// from the top-level Cloud identity: STS/OAuth bearer preferred, else the Cloud
// key/secret, else nil. It takes no resource-scoped parameters, so it cannot
// authenticate with a data-plane credential.
func TestTelemetryAuthFuncScoping(t *testing.T) {
	// Bearer preferred over the Cloud key, using the STS access token (not the raw
	// external OAuth token).
	oauth := &OAuthToken{AccessToken: "external-oauth-token"}
	sts := &STSToken{AccessToken: "sts-token"}
	fn := telemetryAuthFunc("cloud-key", "cloud-secret", oauth, sts)
	if fn == nil {
		t.Fatal("expected a non-nil authFunc for an OAuth/STS identity")
	}
	ctx := fn(context.Background())
	if got, _ := ctx.Value(terraformusagev1.ContextAccessToken).(string); got != "sts-token" {
		t.Errorf("bearer path: ContextAccessToken = %q, want the STS token", got)
	}
	if ctx.Value(terraformusagev1.ContextBasicAuth) != nil {
		t.Errorf("bearer path must not set basic auth")
	}

	// Cloud API key basic auth when no bearer token is present.
	fn = telemetryAuthFunc("cloud-key", "cloud-secret", nil, nil)
	if fn == nil {
		t.Fatal("expected a non-nil authFunc for a Cloud API key identity")
	}
	ctx = fn(context.Background())
	ba, ok := ctx.Value(terraformusagev1.ContextBasicAuth).(terraformusagev1.BasicAuth)
	if !ok || ba.UserName != "cloud-key" || ba.Password != "cloud-secret" {
		t.Errorf("basic path: ContextBasicAuth = %+v (ok=%v), want {cloud-key cloud-secret}", ba, ok)
	}
	if ctx.Value(terraformusagev1.ContextAccessToken) != nil {
		t.Errorf("basic path must not set a bearer token")
	}

	// No top-level identity: nil authFunc (reporting disabled).
	if fn := telemetryAuthFunc("", "", nil, nil); fn != nil {
		t.Errorf("expected a nil authFunc with no top-level identity")
	}
	// A partial Cloud key (secret missing) is not a usable identity.
	if fn := telemetryAuthFunc("cloud-key", "", nil, nil); fn != nil {
		t.Errorf("expected a nil authFunc when the Cloud API secret is missing")
	}
	// An OAuth token without an exchanged STS token is not a usable bearer
	// identity, and with no Cloud key it disables.
	if fn := telemetryAuthFunc("", "", &OAuthToken{AccessToken: "external"}, nil); fn != nil {
		t.Errorf("expected a nil authFunc for an OAuth token with no STS token and no Cloud key")
	}

	// An empty STS AccessToken is not a usable bearer: the bearer path is skipped
	// and it falls through to the Cloud key rather than attaching an empty token.
	fn = telemetryAuthFunc("cloud-key", "cloud-secret", &OAuthToken{AccessToken: "external"}, &STSToken{AccessToken: ""})
	if fn == nil {
		t.Fatal("expected a non-nil authFunc: an empty STS token should fall through to the Cloud key")
	}
	ctx = fn(context.Background())
	if ctx.Value(terraformusagev1.ContextAccessToken) != nil {
		t.Errorf("an empty STS token must not be attached as a bearer")
	}
	if ba, ok := ctx.Value(terraformusagev1.ContextBasicAuth).(terraformusagev1.BasicAuth); !ok || ba.UserName != "cloud-key" {
		t.Errorf("empty STS token should fall through to Cloud key basic auth, got %+v (ok=%v)", ba, ok)
	}
}
