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
	"os"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// restorePublishedTelemetry snapshots the process-global runtime and restores it
// after the test, so a test that publishes a runtime cannot leak into others.
func restorePublishedTelemetry(t *testing.T) {
	t.Helper()
	prev := publishedTelemetry.Load()
	t.Cleanup(func() { publishedTelemetry.Store(prev) })
}

func TestTelemetryOptOut(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		envValue *string // nil = set to empty
		want     bool
	}{
		{"default (prod) endpoint, no env: enabled", defaultCloudEndpoint, nil, false},
		{"staging endpoint, no env: enabled", stagingCloudEndpoint, nil, false},
		{"devel endpoint, no env: enabled", develCloudEndpoint, nil, false},
		{"empty endpoint disables", "", nil, true},
		{"Confluent Cloud for Government endpoint disables", "https://api.confluentgov.com", nil, true},
		{"Confluent Cloud for Government endpoint (CLI form) disables", "https://confluentgov.com/api", nil, true},
		{"trailing slash disables (exact match only)", defaultCloudEndpoint + "/", nil, true},
		{"env var disables on default endpoint", defaultCloudEndpoint, strptr("1"), true},
		{"env var disables on staging endpoint", stagingCloudEndpoint, strptr("1"), true},
		{"env var disables even with a false-ish value", defaultCloudEndpoint, strptr("false"), true}, // any non-empty value opts out
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != nil {
				t.Setenv(disableProviderAnalyticsEnvVar, *tc.envValue)
			} else {
				t.Setenv(disableProviderAnalyticsEnvVar, "")
			}
			if got := telemetryOptOut(tc.endpoint); got != tc.want {
				t.Errorf("telemetryOptOut(%q) = %v, want %v", tc.endpoint, got, tc.want)
			}
		})
	}
}

func TestPublishedTelemetryReporter_DropsWhenDisabledOrUnset(t *testing.T) {
	restorePublishedTelemetry(t)
	reporter := publishedTelemetryReporter{}

	// Nothing published yet: must not panic and must drop.
	publishedTelemetry.Store(nil)
	reporter.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})

	// Disabled runtime: drop even though a reporter is present.
	rec := &recordingReporter{}
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(true), reporter: rec})
	reporter.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})
	if rec.count() != 0 {
		t.Errorf("disabled runtime must drop; got %d events", rec.count())
	}

	// Enabled runtime with no reporter (defensive): drop, don't panic.
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: nil})
	reporter.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})

	// Enabled runtime with a reporter: forward.
	rec2 := &recordingReporter{}
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec2})
	reporter.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})
	if rec2.count() != 1 {
		t.Errorf("enabled runtime must forward; got %d events", rec2.count())
	}
}

// TestPublishedTelemetryReporter_ConcurrentReads publishes once, then issues many
// concurrent Reports and asserts each one forwards. Concurrent load/store of the
// atomic is covered by TestPublishedTelemetryReporter_ConcurrentPublishAndReport.
func TestPublishedTelemetryReporter_ConcurrentReads(t *testing.T) {
	restorePublishedTelemetry(t)
	rec := &recordingReporter{}
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec})

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			publishedTelemetryReporter{}.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic", Operation: telemetry.OperationRead})
		}()
	}
	wg.Wait()
	if rec.count() != n {
		t.Errorf("expected %d forwarded events, got %d", n, rec.count())
	}
}

// TestPublishedTelemetryReporter_ConcurrentPublishAndReport races a publisher
// against concurrent Reports to exercise the atomic pointer under -race. It
// asserts no count (reads straddle the publish) — only race- and panic-freedom.
func TestPublishedTelemetryReporter_ConcurrentPublishAndReport(t *testing.T) {
	restorePublishedTelemetry(t)

	const readers = 20
	var wg sync.WaitGroup
	wg.Add(readers + 1)

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			publishedTelemetry.Store(&telemetryRuntime{
				config:   telemetry.NewConfig(false),
				reporter: &recordingReporter{},
			})
		}
	}()
	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				publishedTelemetryReporter{}.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})
			}
		}()
	}
	wg.Wait()
}

func TestPublishTelemetryRuntime(t *testing.T) {
	// Every case supplies a top-level identity, so each subtest isolates one gate. The
	// no-identity path lives in telemetry_auth_scope_test.go.
	t.Run("non-default endpoint publishes a disabled runtime", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(context.Background(), "https://mock.local", "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime with no reporter, got %+v", rt)
		}
	})

	t.Run("empty endpoint publishes a disabled runtime", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(context.Background(), "", "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime with no reporter, got %+v", rt)
		}
	})

	t.Run("opt-out publishes a disabled runtime even on the default endpoint with an identity", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "1")
		publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime when opted out, got %+v", rt)
		}
	})

	t.Run("test mode publishes a disabled runtime even on the default endpoint with an identity", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, true)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime in test mode, got %+v", rt)
		}
	})

	t.Run("default endpoint, no opt-out: enabled runtime with a live sink", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		// Truly unset, as for a user who never sets it.
		_ = os.Unsetenv(disableProviderAnalyticsEnvVar)
		publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || rt.config.Disabled {
			t.Fatalf("expected an enabled runtime, got %+v", rt)
		}
		if rt.reporter == nil {
			t.Fatalf("enabled runtime must carry a non-nil reporter")
		}
		// Stop the transport's worker goroutines started for this test.
		if c, ok := rt.reporter.(interface{ Close() }); ok {
			c.Close()
		}
		// The published run ID is stable and matches the process run ID.
		if rt.config.RunID != telemetry.RunID() {
			t.Errorf("published RunID = %q, want the process RunID %q", rt.config.RunID, telemetry.RunID())
		}
	})

	t.Run("staging endpoint, no opt-out: enabled runtime with a live sink", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(context.Background(), stagingCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || rt.config.Disabled || rt.reporter == nil {
			t.Fatalf("expected an enabled runtime on the staging endpoint, got %+v", rt)
		}
		// Stop the transport's worker goroutines started for this test.
		if c, ok := rt.reporter.(interface{ Close() }); ok {
			c.Close()
		}
	})
}

// TestPublishedGate_EndToEndThroughWrapper drives a real wrapped Create through
// the same reporter wiring New() installs, and checks the published opt-out state
// controls emission: a disabled runtime emits nothing even with a sink present,
// an enabled runtime forwards exactly one event.
func TestPublishedGate_EndToEndThroughWrapper(t *testing.T) {
	restorePublishedTelemetry(t)

	rec := &recordingReporter{}
	r := newTestResource()
	// Wire the reporter exactly as New() does.
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, telemetryWrapConfig{
		reporter:         publishedTelemetryReporter{},
		providerVersion:  "9.9.9-test",
		terraformVersion: func() string { return "1.7.0-test" },
	})

	// Disabled runtime: a real wrapped Create must emit nothing even though a
	// sink is present behind the gate.
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(true), reporter: rec})
	_ = r.CreateContext(context.Background(), nil, nil)
	if rec.count() != 0 {
		t.Fatalf("disabled runtime must emit zero telemetry through the wrapper; got %d", rec.count())
	}

	// Enabled runtime: the same wrapped call now forwards exactly one event.
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec})
	_ = r.CreateContext(context.Background(), nil, nil)
	if rec.count() != 1 {
		t.Fatalf("enabled runtime must forward exactly one event through the wrapper; got %d", rec.count())
	}
}

// TestDefaultCloudEndpointMatchesSchemaDefault keeps defaultCloudEndpoint in sync
// with the provider's "endpoint" schema default. Reporting is enabled only for
// endpoints in the allowlist, so if the two drifted, telemetry would silently
// disable on the production endpoint.
func TestDefaultCloudEndpointMatchesSchemaDefault(t *testing.T) {
	p := New(testVersion, "")()
	got, ok := p.Schema["endpoint"].Default.(string)
	if !ok {
		t.Fatalf("provider \"endpoint\" schema default is not a string: %T", p.Schema["endpoint"].Default)
	}
	if got != defaultCloudEndpoint {
		t.Errorf("provider \"endpoint\" schema default = %q, want defaultCloudEndpoint %q; the opt-out gate misfires if these diverge", got, defaultCloudEndpoint)
	}
}

// TestTelemetryDisabledForTestMode checks that hermetic acceptance runs suppress
// telemetry while live-production runs may emit.
func TestTelemetryDisabledForTestMode(t *testing.T) {
	tests := []struct {
		name       string
		acceptance bool
		live       bool
		want       bool
	}{
		{"not a test run: not suppressed", false, false, false},
		{"hermetic acceptance: suppressed", true, false, true},
		{"live production: not suppressed", true, true, false},
		{"live flag alone: not suppressed", false, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := telemetryDisabledForTestMode(tc.acceptance, tc.live); got != tc.want {
				t.Errorf("telemetryDisabledForTestMode(%v, %v) = %v, want %v", tc.acceptance, tc.live, got, tc.want)
			}
		})
	}
}

// TestProviderConfigureWiresTelemetryGate checks that providerConfigure passes the
// configured endpoint and the test mode to the telemetry gate.
func TestProviderConfigureWiresTelemetryGate(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string // empty uses the schema default
		acc, prod   string
		wantEnabled bool
	}{
		{"default endpoint: enabled", "", "", "", true},
		{"Confluent Cloud for Government endpoint: disabled", "https://api.confluentgov.com", "", "", false},
		{"hermetic acceptance test: disabled", "", "1", "", false},
		{"live production test: enabled", "", "1", "1", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restorePublishedTelemetry(t)
			publishedTelemetry.Store(nil)
			t.Setenv(disableProviderAnalyticsEnvVar, "")
			t.Setenv("TF_ACC", tc.acc)
			t.Setenv("TF_ACC_PROD", tc.prod)
			raw := map[string]interface{}{"cloud_api_key": "test-key", "cloud_api_secret": "test-secret"}
			if tc.endpoint != "" {
				raw["endpoint"] = tc.endpoint
			}
			p := New("test", "")()
			d := schema.TestResourceDataRaw(t, p.Schema, raw)
			if _, diags := providerConfigure(context.Background(), d, p, "test", ""); diags.HasError() {
				t.Fatalf("providerConfigure failed: %v", diags)
			}
			rt := publishedTelemetry.Load()
			if rt == nil {
				t.Fatal("providerConfigure published no telemetry runtime")
			}
			// Stop the transport an enabled runtime starts.
			if c, ok := rt.reporter.(interface{ Close() }); ok {
				c.Close()
			}
			if enabled := !rt.config.Disabled && rt.reporter != nil; enabled != tc.wantEnabled {
				t.Errorf("reporting enabled = %v, want %v", enabled, tc.wantEnabled)
			}
		})
	}
}

// closeRecordingReporter is a telemetryReporter that records whether Close was called.
type closeRecordingReporter struct {
	closed bool
}

func (c *closeRecordingReporter) Report(telemetry.Usage) {}
func (c *closeRecordingReporter) Close()                 { c.closed = true }

// TestPublishTelemetryRuntime_ClosesPriorTransport checks that reconfiguring closes
// the previously published reporter, so repeated in-process configuration does not
// leak transport workers.
func TestPublishTelemetryRuntime_ClosesPriorTransport(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	prior := &closeRecordingReporter{}
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: prior})

	publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)

	if !prior.closed {
		t.Error("prior reporter was not closed on reconfiguration")
	}
	// Stop the transport this call started.
	if rt := publishedTelemetry.Load(); rt != nil {
		if closer, ok := rt.reporter.(interface{ Close() }); ok {
			closer.Close()
		}
	}
}

func strptr(s string) *string { return &s }
