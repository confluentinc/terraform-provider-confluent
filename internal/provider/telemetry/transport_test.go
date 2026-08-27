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

// TestTransport_ReportNonBlockingAndDropsWhenFull proves Report never blocks the
// caller and that a saturated pool drops events immediately rather than queuing
// unboundedly. With 2 workers stuck in Post and a queue depth of 2, capacity is
// exactly 4: two events reach a worker, two sit in the queue, and everything
// beyond that is dropped.
func TestTransport_ReportNonBlockingAndDropsWhenFull(t *testing.T) {
	release := make(chan struct{})
	fp := newFakePoster(func(ctx context.Context, _ Usage) error {
		<-release // block every worker until the test releases them
		return nil
	})
	tr := newTransport(fp, context.Background(), 2 /*workers*/, 2 /*queueDepth*/, time.Minute)
	defer tr.Close()

	// Fill both workers first, deterministically: fire one, wait for a worker to
	// enter Post, repeat. Now both workers are blocked in Post.
	tr.Report(sampleUsage())
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 2, 2*time.Second)

	if got := fp.calls.Load(); got != 2 {
		t.Fatalf("expected 2 in-flight Posts, got %d", got)
	}

	// Fire 8 more. Two fit the queue; six must be dropped. None of these can
	// enter Post because both workers are blocked — Report must return anyway.
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

	// Release the workers; the two queued events now drain (calls -> 4). The six
	// dropped events never entered, so calls must settle at exactly 4.
	close(release)
	waitSignals(t, fp.entered, 2, 2*time.Second)
	if got := fp.calls.Load(); got != 4 {
		t.Fatalf("expected 4 total Posts (2 in-flight + 2 queued, 6 dropped), got %d", got)
	}
	// Give any erroneous extra deliveries a chance to appear, then confirm none did.
	time.Sleep(50 * time.Millisecond)
	if got := fp.calls.Load(); got != 4 {
		t.Fatalf("dropped events must never be delivered; calls grew to %d", got)
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
		// The first event blocks until its context deadline fires; later events
		// return immediately, proving the worker recovered.
		select {
		case first <- struct{}{}:
			<-ctx.Done()
			return ctx.Err()
		default:
			return nil
		}
	})
	// One worker, so recovery is observable: if the worker wedged on the timed-out
	// send, the second event could never be delivered.
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
	// A retry would produce a second call; give it a window and confirm it stays 1.
	time.Sleep(100 * time.Millisecond)
	if got := fp.calls.Load(); got != 1 {
		t.Fatalf("failed send must not be retried; got %d calls", got)
	}
}
