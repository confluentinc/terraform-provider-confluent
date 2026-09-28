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
	"bytes"
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// capWarnings returns the cap-warning entries logged to a tflogtest root logger.
func capWarnings(t *testing.T, logs *bytes.Buffer) []map[string]interface{} {
	t.Helper()
	entries, err := tflogtest.MultilineJSONDecode(logs)
	if err != nil {
		t.Fatalf("decoding log output: %v", err)
	}
	var warnings []map[string]interface{}
	for _, e := range entries {
		msg, _ := e["@message"].(string)
		if e["@level"] == "warn" && strings.Contains(msg, "event cap reached") {
			warnings = append(warnings, e)
		}
	}
	return warnings
}

// forwardedSequences returns the sequence numbers a recordingReporter received.
func forwardedSequences(rec *recordingReporter) []int64 {
	var seqs []int64
	for _, u := range rec.snapshot() {
		seqs = append(seqs, u.Sequence)
	}
	return seqs
}

func TestPublishedTelemetryReporter_EventCap(t *testing.T) {
	t.Run("drops events past the cap and warns once", func(t *testing.T) {
		restorePublishedTelemetry(t)
		var logs bytes.Buffer
		rec := &recordingReporter{}
		publishedTelemetry.Store(&telemetryRuntime{
			config:    telemetry.NewConfig(false),
			reporter:  rec,
			maxEvents: 3,
			logCtx:    tflogtest.RootLogger(context.Background(), &logs),
		})

		for seq := int64(1); seq <= 6; seq++ {
			publishedTelemetryReporter{}.Report(telemetry.Usage{Sequence: seq})
		}

		if got := forwardedSequences(rec); !reflect.DeepEqual(got, []int64{1, 2, 3}) {
			t.Errorf("forwarded sequences = %v, want [1 2 3]", got)
		}
		warnings := capWarnings(t, &logs)
		if len(warnings) != 1 {
			t.Fatalf("cap warning logged %d times, want once", len(warnings))
		}
		if got := warnings[0]["max_events"]; got != float64(3) {
			t.Errorf("cap warning max_events = %v, want 3", got)
		}
	})

	t.Run("zero cap leaves the run uncapped", func(t *testing.T) {
		restorePublishedTelemetry(t)
		rec := &recordingReporter{}
		publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: rec})

		publishedTelemetryReporter{}.Report(telemetry.Usage{Sequence: math.MaxInt32})

		if rec.count() != 1 {
			t.Errorf("uncapped runtime forwarded %d events, want 1", rec.count())
		}
	})

	t.Run("disabled runtime neither forwards nor warns past the cap", func(t *testing.T) {
		restorePublishedTelemetry(t)
		var logs bytes.Buffer
		rec := &recordingReporter{}
		publishedTelemetry.Store(&telemetryRuntime{
			config:    telemetry.NewConfig(true),
			reporter:  rec,
			maxEvents: 1,
			logCtx:    tflogtest.RootLogger(context.Background(), &logs),
		})

		publishedTelemetryReporter{}.Report(telemetry.Usage{Sequence: 2})

		if rec.count() != 0 {
			t.Errorf("disabled runtime forwarded %d events, want 0", rec.count())
		}
		if warnings := capWarnings(t, &logs); len(warnings) != 0 {
			t.Errorf("disabled runtime logged %d cap warnings, want 0", len(warnings))
		}
	})
}

func TestTelemetryMaxEventsPerRun(t *testing.T) {
	if defaultMaxEventsPerRun != 10000 {
		t.Errorf("defaultMaxEventsPerRun = %d, want 10000", defaultMaxEventsPerRun)
	}
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{"unset keeps the default", "", defaultMaxEventsPerRun},
		{"positive override", "25", 25},
		{"max int64 override", "9223372036854775807", math.MaxInt64},
		{"zero keeps the default", "0", defaultMaxEventsPerRun},
		{"negative keeps the default", "-5", defaultMaxEventsPerRun},
		{"non-numeric keeps the default", "lots", defaultMaxEventsPerRun},
		{"out of range keeps the default", "9223372036854775808", defaultMaxEventsPerRun},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(maxEventsPerRunEnvVar, tc.value)
			if got := telemetryMaxEventsPerRun(); got != tc.want {
				t.Errorf("telemetryMaxEventsPerRun() with %q = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

// TestPublishTelemetryRuntime_SetsEventCap checks that configuration publishes the
// default cap, or the env override, with a logger for the cap warning.
func TestPublishTelemetryRuntime_SetsEventCap(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{"default cap", "", defaultMaxEventsPerRun},
		{"env override", "42", 42},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restorePublishedTelemetry(t)
			t.Setenv(disableProviderAnalyticsEnvVar, "")
			t.Setenv(previewProviderAnalyticsEnvVar, "1")
			t.Setenv(maxEventsPerRunEnvVar, tc.value)

			publishTelemetryRuntime(context.Background(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)

			rt := publishedTelemetry.Load()
			if rt == nil || rt.config.Disabled || rt.reporter == nil {
				t.Fatalf("expected an enabled runtime, got %+v", rt)
			}
			if c, ok := rt.reporter.(interface{ Close() }); ok {
				t.Cleanup(c.Close)
			}
			if rt.maxEvents != tc.want {
				t.Errorf("published maxEvents = %d, want %d", rt.maxEvents, tc.want)
			}
			if rt.logCtx == nil {
				t.Error("published runtime has no logger context for the cap warning")
			}
		})
	}
}

// TestTelemetryEventCap_SendsExactlyNRequests drives n+1 operations through a
// New()-wrapped resource, the real transport, and the SDK poster, and checks a
// stub endpoint receives exactly n requests.
func TestTelemetryEventCap_SendsExactlyNRequests(t *testing.T) {
	restorePublishedTelemetry(t)
	const n = 5

	var requests atomic.Int64
	arrived := make(chan struct{}, n+1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/terraform-usage/v1/usages" {
			requests.Add(1)
			arrived <- struct{}{}
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	transport := telemetry.NewTransport(telemetry.NewSDKPoster(server.URL, server.Client(), "ua", nil), context.Background())
	defer transport.Close()

	// Cap the run n sequence numbers past the current one, so the next n
	// operations are reported and the one after is not.
	var logs bytes.Buffer
	base := telemetry.NextSequence()
	publishedTelemetry.Store(&telemetryRuntime{
		config:    telemetry.NewConfig(false),
		reporter:  transport,
		maxEvents: base + n,
		logCtx:    tflogtest.RootLogger(context.Background(), &logs),
	})

	r := New(testVersion, "")().ResourcesMap["confluent_environment"]
	for i := 1; i <= n; i++ {
		// nil args fail the read without a network call; the wrapper still reports.
		_ = r.ReadContext(context.Background(), nil, nil)
		// Wait for each request so the shallow transport queue never drops one.
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			t.Fatalf("request %d/%d never reached the stub", i, n)
		}
	}
	_ = r.ReadContext(context.Background(), nil, nil) // operation n+1, past the cap

	select {
	case <-arrived:
		t.Fatalf("an event past the cap reached the stub (%d requests)", requests.Load())
	case <-time.After(200 * time.Millisecond):
	}
	if got := requests.Load(); got != n {
		t.Errorf("stub received %d requests, want exactly %d", got, n)
	}
	if warnings := capWarnings(t, &logs); len(warnings) != 1 {
		t.Errorf("cap warning logged %d times, want once", len(warnings))
	}
}

// TestTelemetryEventCap_ConcurrentOperations runs wrapped operations from 10
// goroutines, Terraform's default parallelism, and checks that exactly the first n
// sequence numbers are forwarded and the warning fires once (run with -race).
func TestTelemetryEventCap_ConcurrentOperations(t *testing.T) {
	restorePublishedTelemetry(t)
	const (
		parallelism  = 10
		opsPerWorker = 30
		n            = 100
	)

	r := newTestResource()
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(publishedTelemetryReporter{}))

	var logs bytes.Buffer
	rec := &recordingReporter{}
	base := telemetry.NextSequence()
	publishedTelemetry.Store(&telemetryRuntime{
		config:    telemetry.NewConfig(false),
		reporter:  rec,
		maxEvents: base + n,
		logCtx:    tflogtest.RootLogger(context.Background(), &logs),
	})

	var wg sync.WaitGroup
	for w := 0; w < parallelism; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				_ = r.ReadContext(context.Background(), nil, nil)
			}
		}()
	}
	wg.Wait()

	seqs := forwardedSequences(rec)
	if len(seqs) != n {
		t.Fatalf("forwarded %d events, want exactly %d", len(seqs), n)
	}
	seen := make(map[int64]bool, n)
	for _, s := range seqs {
		if s <= base || s > base+n || seen[s] {
			t.Errorf("forwarded sequence %d is a duplicate or outside the first %d of the run", s, n)
		}
		seen[s] = true
	}
	if warnings := capWarnings(t, &logs); len(warnings) != 1 {
		t.Errorf("cap warning logged %d times, want once", len(warnings))
	}
}
