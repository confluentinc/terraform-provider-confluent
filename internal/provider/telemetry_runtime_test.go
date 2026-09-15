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
		envValue *string // nil = unset
		want     bool
	}{
		{"default endpoint, no env: enabled", defaultCloudEndpoint, nil, false},
		{"empty endpoint disables (only prod endpoint enables)", "", nil, true},
		{"non-default (gov) endpoint disables", "https://api.confluent-gov.cloud", nil, true},
		{"env var disables on default endpoint", defaultCloudEndpoint, strptr("1"), true},
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
	// A top-level Cloud identity is supplied in every case so that endpoint/env
	// (or, below, test mode) is the only factor under test — never a missing
	// identity. The no-identity path is covered in telemetry_auth_scope_test.go.
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

	t.Run("env var publishes a disabled runtime even on the default endpoint", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "1")
		publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime, got %+v", rt)
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

	t.Run("default endpoint, no env var: enabled runtime with a live sink", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
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
// with the provider's "endpoint" schema default. The gate enables reporting only
// on an exact match, so if the two drifted, telemetry would silently disable on
// the production endpoint.
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

func strptr(s string) *string { return &s }
