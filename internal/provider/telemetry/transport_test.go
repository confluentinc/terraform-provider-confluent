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
	"sync"
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

// waitSettled waits until nothing is queued or being sent, or fails after timeout.
// Unlike Drain it also waits after failed sends.
func waitSettled(t *testing.T, tr *Transport, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		tr.mu.Lock()
		pending := tr.pending
		tr.mu.Unlock()
		if pending == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%d events still unsent after %s", pending, timeout)
		}
		time.Sleep(time.Millisecond)
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

// TestTransport_DrainReturnsAtOnceWhenIdle confirms Drain does not wait when
// nothing is queued or being sent, before and after a delivery.
func TestTransport_DrainReturnsAtOnceWhenIdle(t *testing.T) {
	fp := newFakePoster(nil)
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	start := time.Now()
	if !tr.Drain(2 * time.Second) {
		t.Fatal("Drain on a new transport reported unsent events")
	}
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	if !tr.Drain(2 * time.Second) {
		t.Fatal("Drain after the only event was sent reported unsent events")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("Drain waited %s with nothing unsent", elapsed)
	}
}

// TestTransport_DrainWaitsForQueuedAndInFlightEvents confirms Drain returns only
// once every queued and in-flight event has been sent.
func TestTransport_DrainWaitsForQueuedAndInFlightEvents(t *testing.T) {
	release := make(chan struct{})
	var finished atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		<-release
		finished.Add(1)
		return nil
	})
	// One worker: one event in flight, two queued behind it.
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	for i := 0; i < 3; i++ {
		tr.Report(sampleUsage())
	}
	waitSignals(t, fp.entered, 1, 2*time.Second)

	drained := make(chan bool, 1)
	go func() { drained <- tr.Drain(2 * time.Second) }()
	select {
	case <-drained:
		t.Fatal("Drain returned while events were still unsent")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	select {
	case ok := <-drained:
		if !ok {
			t.Fatal("Drain reported unsent events after all were sent")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Drain did not return after all events were sent")
	}
	if got := finished.Load(); got != 3 {
		t.Fatalf("expected 3 finished sends before Drain returned, got %d", got)
	}
}

// TestTransport_DrainStopsAtTimeout confirms Drain gives up at its timeout when
// a send never finishes. The timeout counts from when the event was queued.
func TestTransport_DrainStopsAtTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	fp := newFakePoster(func(context.Context, Usage) error {
		<-release
		return nil
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	const timeout = 100 * time.Millisecond
	start := time.Now()
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	if tr.Drain(timeout) {
		t.Fatal("Drain reported nothing unsent while a send was still running")
	}
	if elapsed := time.Since(start); elapsed < timeout || elapsed > timeout+time.Second {
		t.Fatalf("Drain returned after %s, want about %s", elapsed, timeout)
	}
}

// TestTransport_DrainSkipsSendsStuckLongerThanTimeout confirms Drain returns at
// once when nothing has settled for longer than its timeout, so a backend that
// stopped answering is not waited on again.
func TestTransport_DrainSkipsSendsStuckLongerThanTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	fp := newFakePoster(func(context.Context, Usage) error {
		<-release
		return nil
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	const timeout = 200 * time.Millisecond
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	time.Sleep(2 * timeout)
	tr.Report(sampleUsage()) // queued behind the stuck send; must not restart the wait

	start := time.Now()
	if tr.Drain(timeout) {
		t.Fatal("Drain reported nothing unsent while a send was stuck")
	}
	if elapsed := time.Since(start); elapsed > timeout/2 {
		t.Fatalf("Drain waited %s on sends stuck for longer than its timeout", elapsed)
	}
}

// TestTransport_DrainWaitsOnlyTheRestOfTheTimeout confirms Drain counts its
// timeout from the transport's last activity, not from the call.
func TestTransport_DrainWaitsOnlyTheRestOfTheTimeout(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	fp := newFakePoster(func(context.Context, Usage) error {
		<-release
		return nil
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	const timeout = 200 * time.Millisecond
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	time.Sleep(timeout * 3 / 4)

	start := time.Now()
	if tr.Drain(timeout) {
		t.Fatal("Drain reported nothing unsent while a send was stuck")
	}
	if elapsed := time.Since(start); elapsed > timeout*3/4 {
		t.Fatalf("Drain waited %s; it should wait only the rest of its %s timeout", elapsed, timeout)
	}
}

// TestTransport_DrainRestartsAfterASettle confirms a send that settles makes the
// transport active again, so Drain waits for the event queued behind it.
func TestTransport_DrainRestartsAfterASettle(t *testing.T) {
	const timeout = 200 * time.Millisecond
	var n atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		if n.Add(1) == 1 {
			time.Sleep(2 * timeout) // a slow send that still succeeds
		} else {
			time.Sleep(timeout / 10)
		}
		return nil
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	tr.Report(sampleUsage())
	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 2, 2*time.Second) // the first has settled
	if !tr.Drain(timeout) {
		t.Fatal("Drain did not wait for the event queued behind a send that just settled")
	}
}

// TestTransport_DrainCountsFailedAndPanickedSends confirms a failed or panicking
// send still counts as settled, so nothing is left pending.
func TestTransport_DrainCountsFailedAndPanickedSends(t *testing.T) {
	var n atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		switch n.Add(1) {
		case 2:
			return errors.New("boom")
		case 3:
			panic("post boom")
		default:
			return nil
		}
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	// The last send panics, so Drain must report nothing pending despite the failure.
	for i := 0; i < 3; i++ {
		tr.Report(sampleUsage())
	}
	waitSettled(t, tr, 2*time.Second)
	if !tr.Drain(time.Second) {
		t.Fatal("Drain reported unsent events after every send settled")
	}
	if got := fp.calls.Load(); got != 3 {
		t.Fatalf("expected 3 sends, got %d", got)
	}
}

// TestTransport_DrainSkipsAfterAFailedSend confirms Drain does not wait when the
// most recently settled send failed or panicked.
func TestTransport_DrainSkipsAfterAFailedSend(t *testing.T) {
	for _, tc := range []struct {
		name string
		fail func() error
	}{
		{"error", func() error { return errors.New("boom") }},
		{"panic", func() error { panic("post boom") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			release := make(chan struct{})
			defer close(release)
			var n atomic.Int64
			fp := newFakePoster(func(context.Context, Usage) error {
				if n.Add(1) == 1 {
					return tc.fail()
				}
				<-release
				return nil
			})
			tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
			defer tr.Close()

			tr.Report(sampleUsage()) // fails
			tr.Report(sampleUsage()) // then hangs
			waitSignals(t, fp.entered, 2, 2*time.Second)

			start := time.Now()
			if tr.Drain(time.Second) {
				t.Fatal("Drain reported nothing unsent while a send was stuck")
			}
			if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
				t.Fatalf("Drain waited %s after a failed send", elapsed)
			}
		})
	}
}

// TestTransport_DrainWaitsAgainAfterASuccessfulSend confirms a successful send
// after a failure makes Drain wait again.
func TestTransport_DrainWaitsAgainAfterASuccessfulSend(t *testing.T) {
	release := make(chan struct{})
	var n atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		switch n.Add(1) {
		case 1:
			return errors.New("boom")
		case 2:
			return nil
		}
		<-release
		return nil
	})
	tr := newTransport(fp, context.Background(), 1, 4, time.Minute)
	defer tr.Close()

	for i := 0; i < 3; i++ {
		tr.Report(sampleUsage())
	}
	waitSignals(t, fp.entered, 3, 2*time.Second)
	time.AfterFunc(50*time.Millisecond, func() { close(release) })
	if !tr.Drain(2 * time.Second) {
		t.Fatal("Drain did not wait for the last send after a successful one")
	}
}

// TestTransport_DrainReturnsWithinTimeoutWhileSendsKeepSettling confirms the
// deadline is fixed when Drain is called: sends that keep settling don't extend it.
func TestTransport_DrainReturnsWithinTimeoutWhileSendsKeepSettling(t *testing.T) {
	// 4 workers at 20 ms per send fall behind one event per millisecond, so the
	// transport stays busy and a send settles every few milliseconds.
	tr := newTransport(postFunc(func(context.Context, Usage) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	}), context.Background(), 4, 8, time.Minute)
	defer tr.Close()

	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				tr.Report(sampleUsage())
				time.Sleep(time.Millisecond)
			}
		}
	}()
	defer func() {
		close(stop)
		wg.Wait()
	}()
	const timeout = 300 * time.Millisecond
	time.Sleep(50 * time.Millisecond)

	start := time.Now()
	tr.Drain(timeout)
	if elapsed := time.Since(start); elapsed > timeout*3/2 {
		t.Fatalf("Drain returned after %s; its timeout was %s", elapsed, timeout)
	}
}

// TestTransport_DrainIgnoresDroppedEvents confirms events dropped by a full queue
// are not waited for.
func TestTransport_DrainIgnoresDroppedEvents(t *testing.T) {
	release := make(chan struct{})
	fp := newFakePoster(func(context.Context, Usage) error {
		<-release
		return nil
	})
	// 1 worker + queue 1: two events fit, the other eight are dropped.
	tr := newTransport(fp, context.Background(), 1, 1, time.Minute)
	defer tr.Close()

	tr.Report(sampleUsage())
	waitSignals(t, fp.entered, 1, 2*time.Second)
	for i := 0; i < 9; i++ {
		tr.Report(sampleUsage())
	}
	close(release)
	if !tr.Drain(2 * time.Second) {
		t.Fatal("Drain waited for events the full queue dropped")
	}
	if got := fp.calls.Load(); got != 2 {
		t.Fatalf("expected 2 sends, got %d", got)
	}
}

// TestTransport_DrainDuringConcurrentReports confirms Drain is safe to call while
// other goroutines are still reporting (data races show only under -race).
func TestTransport_DrainDuringConcurrentReports(t *testing.T) {
	tr := newTransport(noopPoster{}, context.Background(), 4, 8, time.Minute)
	defer tr.Close()

	stop := make(chan struct{})
	reporters := make(chan struct{})
	for i := 0; i < 4; i++ {
		go func() {
			defer func() { reporters <- struct{}{} }()
			for {
				select {
				case <-stop:
					return
				default:
					tr.Report(sampleUsage())
				}
			}
		}()
	}
	for i := 0; i < 50; i++ {
		tr.Drain(time.Millisecond)
	}
	close(stop)
	for i := 0; i < 4; i++ {
		<-reporters
	}
	if !tr.Drain(2 * time.Second) {
		t.Fatal("Drain did not settle after reporting stopped")
	}
}

// TestTransport_QueuedEventsAreAlwaysPending samples the accounting while
// reporters, a slow or failing Poster, and Drain run concurrently: every queued
// event is counted, and idle is closed exactly when nothing is pending.
func TestTransport_QueuedEventsAreAlwaysPending(t *testing.T) {
	var n atomic.Int64
	poster := postFunc(func(context.Context, Usage) error {
		switch n.Add(1) % 10 {
		case 0:
			panic("post boom")
		case 1:
			return errors.New("boom")
		case 2:
			time.Sleep(100 * time.Microsecond)
		}
		return nil
	})
	for round := 0; round < 20; round++ {
		tr := newTransport(poster, context.Background(), 4, 8, time.Minute)
		stop := make(chan struct{})
		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						tr.Report(sampleUsage())
					}
				}
			}()
		}
		deadline := time.Now().Add(5 * time.Millisecond)
		for time.Now().Before(deadline) {
			tr.mu.Lock()
			pending, queued := tr.pending, len(tr.queue)
			idle := false
			select {
			case <-tr.idle:
				idle = true
			default:
			}
			tr.mu.Unlock()
			if queued > pending || idle != (pending == 0) {
				t.Fatalf("round %d: queued=%d pending=%d idle=%v", round, queued, pending, idle)
			}
			tr.Drain(0)
		}
		close(stop)
		wg.Wait()
		waitSettled(t, tr, 2*time.Second)
		tr.Close()
	}
}

// postFunc adapts a function to Poster.
type postFunc func(context.Context, Usage) error

func (f postFunc) Post(ctx context.Context, u Usage) error { return f(ctx, u) }
