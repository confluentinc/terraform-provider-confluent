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
	"sync/atomic"
	"testing"
	"time"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// postFunc adapts a function to telemetry.Poster.
type postFunc func(context.Context, telemetry.Usage) error

func (f postFunc) Post(ctx context.Context, u telemetry.Usage) error { return f(ctx, u) }

// TestTelemetryExitTimeout pins the exit wait to 500 ms, well under the 2 s
// Terraform allows a provider to exit before killing it.
func TestTelemetryExitTimeout(t *testing.T) {
	if telemetryExitTimeout != 500*time.Millisecond {
		t.Fatalf("telemetryExitTimeout = %s, want 500ms", telemetryExitTimeout)
	}
}

// TestFlushTelemetryReturnsAtOnceWithoutATransport covers the runtimes that have
// nothing to send: none published, reporting disabled, and a reporter that can't
// drain.
func TestFlushTelemetryReturnsAtOnceWithoutATransport(t *testing.T) {
	tests := []struct {
		name string
		rt   *telemetryRuntime
	}{
		{"nothing published", nil},
		{"reporting disabled", &telemetryRuntime{config: telemetry.NewConfig(true)}},
		{"reporter without Drain", &telemetryRuntime{config: telemetry.NewConfig(false), reporter: &recordingReporter{}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restorePublishedTelemetry(t)
			publishedTelemetry.Store(tc.rt)

			start := time.Now()
			FlushTelemetry()
			if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
				t.Fatalf("FlushTelemetry waited %s with nothing to send", elapsed)
			}
		})
	}
}

// TestFlushTelemetrySendsUnsentEvents confirms FlushTelemetry waits for events
// still being sent, and returns as soon as they are.
func TestFlushTelemetrySendsUnsentEvents(t *testing.T) {
	restorePublishedTelemetry(t)
	var sent atomic.Int64
	tr := telemetry.NewTransport(postFunc(func(context.Context, telemetry.Usage) error {
		time.Sleep(20 * time.Millisecond)
		sent.Add(1)
		return nil
	}), context.Background())
	t.Cleanup(tr.Close)
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: tr})

	const n = 6
	for i := 0; i < n; i++ {
		publishedTelemetryReporter{}.Report(telemetry.Usage{Operation: telemetry.OperationCreate})
	}
	start := time.Now()
	FlushTelemetry()
	elapsed := time.Since(start)

	if got := sent.Load(); got != n {
		t.Fatalf("FlushTelemetry returned with %d of %d events sent", got, n)
	}
	if elapsed > telemetryExitTimeout*3/4 {
		t.Fatalf("FlushTelemetry took %s; it should return once the events are sent", elapsed)
	}
}

// TestFlushTelemetryStopsAtTheExitTimeout confirms a backend that never answers
// holds the process for telemetryExitTimeout and no longer.
func TestFlushTelemetryStopsAtTheExitTimeout(t *testing.T) {
	restorePublishedTelemetry(t)
	hang := make(chan struct{})
	t.Cleanup(func() { close(hang) })
	tr := telemetry.NewTransport(postFunc(func(ctx context.Context, _ telemetry.Usage) error {
		select {
		case <-hang:
		case <-ctx.Done():
		}
		return ctx.Err()
	}), context.Background())
	t.Cleanup(tr.Close)
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: tr})

	start := time.Now()
	publishedTelemetryReporter{}.Report(telemetry.Usage{Operation: telemetry.OperationCreate})
	FlushTelemetry()
	elapsed := time.Since(start)

	if elapsed < telemetryExitTimeout || elapsed > telemetryExitTimeout+400*time.Millisecond {
		t.Fatalf("FlushTelemetry returned after %s, want about %s", elapsed, telemetryExitTimeout)
	}
}
