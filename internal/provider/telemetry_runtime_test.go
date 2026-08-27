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
	"sync"
	"testing"

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
		{"default endpoint, no env", defaultCloudEndpoint, nil, false},
		{"empty endpoint disables (only prod endpoint enables)", "", nil, true},
		{"non-default endpoint disables", "https://api.confluent-gov.cloud", nil, true},
		{"env var disables on default endpoint", defaultCloudEndpoint, strptr("1"), true},
		{"env var disables even with true-ish value", defaultCloudEndpoint, strptr("false"), true}, // any non-empty value opts out
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

	// Enabled runtime: forward.
	rec2 := &recordingReporter{}
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec2})
	reporter.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic"})
	if rec2.count() != 1 {
		t.Errorf("enabled runtime must forward; got %d events", rec2.count())
	}
}

// TestPublishedTelemetryReporter_ConcurrentSafe runs many concurrent Reports
// against a published runtime; run with -race to catch unsafe publication.
func TestPublishedTelemetryReporter_ConcurrentSafe(t *testing.T) {
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

func TestPublishTelemetryRuntime(t *testing.T) {
	t.Run("non-default endpoint publishes a disabled runtime", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(t.Context(), "https://mock.local", "ua", "key", "secret", nil, nil)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime with no reporter, got %+v", rt)
		}
	})

	t.Run("env var publishes a disabled runtime", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "1")
		publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "key", "secret", nil, nil)
		rt := publishedTelemetry.Load()
		if rt == nil || !rt.config.Disabled || rt.reporter != nil {
			t.Fatalf("expected a disabled runtime, got %+v", rt)
		}
	})

	t.Run("default endpoint with a cloud key publishes an enabled transport", func(t *testing.T) {
		restorePublishedTelemetry(t)
		t.Setenv(disableProviderAnalyticsEnvVar, "")
		publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "key", "secret", nil, nil)
		rt := publishedTelemetry.Load()
		if rt == nil || rt.config.Disabled {
			t.Fatalf("expected an enabled runtime, got %+v", rt)
		}
		transport, ok := rt.reporter.(*telemetry.Transport)
		if !ok || transport == nil {
			t.Fatalf("expected a *telemetry.Transport reporter, got %T", rt.reporter)
		}
		transport.Close()
	})
}

func TestTelemetryAuthFunc(t *testing.T) {
	if telemetryAuthFunc("", "", nil, nil) != nil {
		t.Errorf("no top-level identity should yield a nil auth func")
	}
	if telemetryAuthFunc("key", "secret", nil, nil) == nil {
		t.Errorf("a cloud API key should yield an auth func")
	}
	if telemetryAuthFunc("", "", &OAuthToken{AccessToken: "tok"}, nil) == nil {
		t.Errorf("an OAuth token should yield an auth func")
	}
	if telemetryAuthFunc("", "", nil, &STSToken{AccessToken: "tok"}) == nil {
		t.Errorf("an STS token should yield an auth func")
	}
}

func strptr(s string) *string { return &s }
