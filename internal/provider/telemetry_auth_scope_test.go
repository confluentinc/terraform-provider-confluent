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

// TestTelemetryDisabledWithoutTopLevelIdentity covers TFCA-B7's acceptance
// criterion: a provider configured with no top-level Cloud identity (for example
// only resource-scoped Kafka credentials, which are never passed here) reports
// nothing — the runtime is disabled with no transport, so zero telemetry calls
// are made. Reporting with no identity is a no-op, never an error.
func TestTelemetryDisabledWithoutTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	// Default endpoint (would otherwise enable) and not test mode, but no Cloud
	// key and no OAuth/STS token — mirroring a Kafka-only provider configuration.
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

	// With no transport, the late-binding reporter is a no-op: it must not panic
	// and makes zero calls (there is nothing to send through).
	publishedTelemetryReporter{}.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic", Operation: telemetry.OperationCreate})
}

// TestTelemetryEnabledWithTopLevelIdentity is the positive control: the same
// inputs plus a top-level Cloud API key on the default endpoint yield an enabled
// runtime with a live transport. Without this, the negative test above could pass
// simply because nothing ever enables.
func TestTelemetryEnabledWithTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
	rt := publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled || rt.reporter == nil {
		t.Fatalf("expected an enabled runtime with a top-level identity, got %+v", rt)
	}
	// Stop the transport's worker goroutines started for this test.
	if c, ok := rt.reporter.(interface{ Close() }); ok {
		c.Close()
	}
}

// TestTelemetryAuthFuncScoping verifies telemetryAuthFunc reads only the
// top-level Cloud identity and selects the right scheme: an OAuth/STS bearer
// token is preferred over the Cloud API key, the Cloud API key is used when no
// bearer token is present, and no top-level identity yields nil (disabled).
// telemetryAuthFunc takes no Kafka/Schema Registry/Flink/Tableflow parameters, so
// it structurally cannot authenticate with a resource-scoped credential.
func TestTelemetryAuthFuncScoping(t *testing.T) {
	// Bearer preferred: an OAuth/STS identity wins even when a Cloud key is also
	// set, and the attached token is the STS access token (what the provider uses
	// for Cloud APIs), never the raw external OAuth token.
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
}
