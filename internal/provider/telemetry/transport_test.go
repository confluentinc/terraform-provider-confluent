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

package telemetry

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// fakePoster is a controllable Poster for exercising the transport in isolation.
type fakePoster struct {
	calls   atomic.Int64
	entered chan struct{} // one signal per Post entry
	fn      func(ctx context.Context, u Usage) error
}

func newFakePoster(fn func(ctx context.Context, u Usage) error) *fakePoster {
	return &fakePoster{entered: make(chan struct{}, 1024), fn: fn}
}

func (f *fakePoster) Post(ctx context.Context, u Usage) error {
	f.calls.Add(1)
	f.entered <- struct{}{}
	if f.fn != nil {
		return f.fn(ctx, u)
	}
	return nil
}

func sampleUsage() Usage {
	return Usage{
		RunID:             RunID(),
		Sequence:          NextSequence(),
		StartedAt:         time.Now(),
		DurationMs:        1,
		OS:                "darwin",
		Arch:              "arm64",
		ProviderVersion:   "9.9.9",
		TerraformVersion:  "1.7.0",
		ResourceType:      "confluent_kafka_topic",
		Operation:         OperationCreate,
		ChangedAttributes: []string{},
	}
}

// waitSignals reads n signals from ch or fails after timeout.
func waitSignals(t *testing.T, ch <-chan struct{}, n int, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for i := 0; i < n; i++ {
		select {
		case <-ch:
		case <-deadline:
			t.Fatalf("timed out waiting for signal %d/%d", i+1, n)
		}
	}
}

// TestTransport_ReportNonBlockingAndDropsWhenFull asserts Report never blocks and
// a saturated pool drops events (2 workers + queue 2 => capacity 4).
func TestTransport_ReportNonBlockingAndDropsWhenFull(t *testing.T) {
	release := make(chan struct{})
	fp := newFakePoster(func(ctx context.Context, _ Usage) error {
		<-release // block every worker until the test releases them
		return nil
	})
	tr := newTransport(fp, context.Background(), 2 /*workers*/, 2 /*queueDepth*/, time.Minute)
	defer tr.Close()

	// Block both workers in Post.
	tr.Report(sampleUsage())
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 2, 2*time.Second)

	if got := fp.calls.Load(); got != 2 {
		t.Fatalf("expected 2 in-flight Posts, got %d", got)
	}

	// Fire 8 more: 2 fit the queue, 6 are dropped; Report must not block.
	done := make(chan struct{})
	go func() {
		for i := 0; i < 8; i++ {
			tr.Report(sampleUsage())
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Report blocked when the pool was saturated")
	}

	// Still exactly 2 Posts have entered (workers remain blocked).
	if got := fp.calls.Load(); got != 2 {
		t.Fatalf("no new Post should start while workers are blocked, got %d", got)
	}

	// Release the workers; the 2 queued events drain, so calls settle at 4.
	close(release)
	waitSignals(t, fp.entered, 2, 2*time.Second)
	if got := fp.calls.Load(); got != 4 {
		t.Fatalf("expected 4 total Posts (2 in-flight + 2 queued, 6 dropped), got %d", got)
	}
	// The six dropped events must never reach a worker: fail fast if a fifth Post
	// enters within a bounded window, else the window elapsing confirms none did.
	select {
	case <-fp.entered:
		t.Fatalf("dropped events must never be delivered; a 5th Post entered (calls=%d)", fp.calls.Load())
	case <-time.After(50 * time.Millisecond):
	}
}

// TestTransport_DeliversQueuedEvents confirms the happy path: enqueued events
// reach the Poster exactly once each.
func TestTransport_DeliversQueuedEvents(t *testing.T) {
	fp := newFakePoster(nil) // succeed immediately
	tr := newTransport(fp, context.Background(), 2, 8, time.Minute)
	defer tr.Close()

	const n = 5
	for i := 0; i < n; i++ {
		tr.Report(sampleUsage())
	}
	waitSignals(t, fp.entered, n, 2*time.Second)
	if got := fp.calls.Load(); got != n {
		t.Fatalf("expected %d deliveries, got %d", n, got)
	}
}

// TestTransport_AppliesPerReportTimeout asserts each send runs under a bounded
// deadline and that a worker recovers after a send times out (it is not wedged).
func TestTransport_AppliesPerReportTimeout(t *testing.T) {
	var sawDeadline atomic.Bool
	first := make(chan struct{}, 1)
	fp := newFakePoster(func(ctx context.Context, u Usage) error {
		if _, ok := ctx.Deadline(); ok {
			sawDeadline.Store(true)
		}
		// First event blocks until its deadline; later events return immediately.
		select {
		case first <- struct{}{}:
			<-ctx.Done()
			return ctx.Err()
		default:
			return nil
		}
	})
	// One worker, so a wedged send would block the second event.
	tr := newTransport(fp, context.Background(), 1, 4, 50*time.Millisecond)
	defer tr.Close()

	tr.Report(sampleUsage()) // times out after ~50ms
	tr.Report(sampleUsage()) // must be delivered once the worker recovers
	waitSignals(t, fp.entered, 2, 2*time.Second)

	if !sawDeadline.Load() {
		t.Errorf("Post was not given a context with a deadline")
	}
}

// TestTransport_NoRetryOnFailure asserts a failed send is dropped, not retried.
func TestTransport_NoRetryOnFailure(t *testing.T) {
	fp := newFakePoster(func(context.Context, Usage) error {
		return errors.New("boom")
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	// A retry would enter Post again; fail fast if it does within the window.
	select {
	case <-fp.entered:
		t.Fatalf("failed send must not be retried; a 2nd Post entered (calls=%d)", fp.calls.Load())
	case <-time.After(100 * time.Millisecond):
	}
}

// TestTransport_RecoversPosterPanic asserts a panicking Post is contained and the
// worker survives to deliver the next event (an unrecovered panic would crash the
// process).
func TestTransport_RecoversPosterPanic(t *testing.T) {
	var n atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		if n.Add(1) == 1 {
			panic("post boom") // first delivery panics on the worker goroutine
		}
		return nil // later deliveries succeed
	})
	// One worker, so the second delivery only happens if it survived the panic.
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	tr.Report(sampleUsage()) // Post panics; deliver must recover
	tr.Report(sampleUsage()) // must still be delivered by the surviving worker
	waitSignals(t, fp.entered, 2, 2*time.Second)
}

// TestNewTransport_UsesDefaults pins the default pool size, queue depth, and
// per-report timeout the constructor uses.
func TestNewTransport_UsesDefaults(t *testing.T) {
	tr := NewTransport(newFakePoster(nil), context.Background())
	defer tr.Close()

	if got := cap(tr.queue); got != defaultQueueDepth {
		t.Errorf("queue depth = %d, want %d", got, defaultQueueDepth)
	}
	if tr.timeout != defaultPerReportTimeout {
		t.Errorf("per-report timeout = %s, want %s", tr.timeout, defaultPerReportTimeout)
	}
	// Pin the constants to their literal values.
	if defaultWorkers != 4 {
		t.Errorf("defaultWorkers = %d, want 4", defaultWorkers)
	}
	if defaultQueueDepth != 8 {
		t.Errorf("defaultQueueDepth = %d, want 8", defaultQueueDepth)
	}
	if defaultQueueDepth != defaultWorkers*2 {
		t.Errorf("defaultQueueDepth = %d, want workers*2 (=%d) so the queue stays shallow", defaultQueueDepth, defaultWorkers*2)
	}
	if defaultPerReportTimeout != 5*time.Second {
		t.Errorf("defaultPerReportTimeout = %s, want 5s (CLI parity)", defaultPerReportTimeout)
	}
}

// TestNewTransport_NilLogCtx confirms a nil logCtx is tolerated (it falls back to
// a background context) and the default transport still delivers without panicking.
func TestNewTransport_NilLogCtx(t *testing.T) {
	fp := newFakePoster(nil)
	tr := NewTransport(fp, nil)
	defer tr.Close()

	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
}

// TestNewTransport_NilPosterIsSafe confirms a nil Poster is replaced with a no-op
// so a worker never nil-panics.
func TestNewTransport_NilPosterIsSafe(t *testing.T) {
	tr := NewTransport(nil, context.Background())
	defer tr.Close()

	if tr.poster == nil {
		t.Fatal("nil Poster was not replaced with a safe no-op; a worker would nil-panic on Post")
	}
	// Also smoke the delivery path against the substitute.
	tr.Report(sampleUsage())
}

// TestTransport_CloseIsIdempotent confirms Close can be called more than once
// without panicking.
func TestTransport_CloseIsIdempotent(t *testing.T) {
	tr := newTransport(newFakePoster(nil), context.Background(), 1, 1, time.Minute)
	tr.Close()
	tr.Close() // must not panic
}
