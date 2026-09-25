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
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
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

// waitFor polls cond until it holds or fails after timeout.
func waitFor(t *testing.T, cond func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for condition")
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

// TestTransport_StopsOnOverloadStatus asserts a 429 or 5xx response stops the run
// after that one send: the events queued behind it are never sent.
func TestTransport_StopsOnOverloadStatus(t *testing.T) {
	for _, code := range []int{
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		599, // the top of the 5xx range
	} {
		t.Run(strconv.Itoa(code), func(t *testing.T) {
			fp := newFakePoster(func(context.Context, Usage) error {
				return &statusError{code: code}
			})
			// One worker sends the queued events in order.
			tr := newTransport(fp, context.Background(), 1, 8, time.Minute)
			defer tr.Close()

			for i := 0; i < 5; i++ {
				tr.Report(sampleUsage())
			}
			waitSignals(t, fp.entered, 1, 2*time.Second)
			waitFor(t, tr.stopped.Load, 2*time.Second)
			select {
			case <-fp.entered:
				t.Fatalf("a %d must stop the run; a 2nd Post entered (calls=%d)", code, fp.calls.Load())
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

// TestTransport_StopsAfterConsecutiveFailures asserts any other failure stops the
// run only once maxConsecutiveFailures sends in a row have failed: exactly that
// many are sent, and none after.
func TestTransport_StopsAfterConsecutiveFailures(t *testing.T) {
	cases := map[string]func(ctx context.Context) error{
		"transport error": func(context.Context) error { return errors.New("connection refused") },
		"timeout":         func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() },
		"panic":           func(context.Context) error { panic("post boom") },
		"400":             func(context.Context) error { return &statusError{code: http.StatusBadRequest} },
		"401":             func(context.Context) error { return &statusError{code: http.StatusUnauthorized} },
		"403":             func(context.Context) error { return &statusError{code: http.StatusForbidden} },
		"404":             func(context.Context) error { return &statusError{code: http.StatusNotFound} },
		"600":             func(context.Context) error { return &statusError{code: 600} }, // outside 5xx
	}
	for name, fail := range cases {
		t.Run(name, func(t *testing.T) {
			fp := newFakePoster(func(ctx context.Context, _ Usage) error { return fail(ctx) })
			// One worker sends the queued events in order; the short timeout keeps
			// the timeout case fast.
			tr := newTransport(fp, context.Background(), 1, 8, 20*time.Millisecond)
			defer tr.Close()

			for i := 0; i < maxConsecutiveFailures+3; i++ {
				tr.Report(sampleUsage())
			}
			waitSignals(t, fp.entered, maxConsecutiveFailures, 2*time.Second)
			waitFor(t, tr.stopped.Load, 2*time.Second)
			select {
			case <-fp.entered:
				t.Fatalf("the run must stop after %d consecutive failures; Post entered again (calls=%d)", maxConsecutiveFailures, fp.calls.Load())
			case <-time.After(100 * time.Millisecond):
			}
		})
	}
}

// TestTransport_SuccessResetsConsecutiveFailures asserts a success clears the
// failure count, so failures that never reach maxConsecutiveFailures in a row do
// not stop the run.
func TestTransport_SuccessResetsConsecutiveFailures(t *testing.T) {
	var n atomic.Int64
	fp := newFakePoster(func(context.Context, Usage) error {
		// Every maxConsecutiveFailures-th send succeeds, so failures come in runs
		// one short of the limit.
		if n.Add(1)%maxConsecutiveFailures == 0 {
			return nil
		}
		return errors.New("boom")
	})
	tr := newTransport(fp, context.Background(), 1, 16, time.Minute)
	defer tr.Close()

	const events = 3 * maxConsecutiveFailures
	for i := 0; i < events; i++ {
		tr.Report(sampleUsage())
	}
	waitSignals(t, fp.entered, events, 2*time.Second)
	if tr.stopped.Load() {
		t.Fatal("failures separated by successes must not stop the run")
	}
}

// TestTransport_ReportAfterStopIsNotQueued asserts that once the run has stopped,
// Report drops events without queueing them.
func TestTransport_ReportAfterStopIsNotQueued(t *testing.T) {
	// No workers, so anything queued stays visible in the queue.
	tr := newTransport(newFakePoster(nil), context.Background(), 0, 4, time.Minute)
	defer tr.Close()

	tr.stopped.Store(true)
	tr.Report(sampleUsage())
	if n := len(tr.queue); n != 0 {
		t.Fatalf("queued %d events after the run stopped, want 0", n)
	}
}

// TestTransport_CapsEventsPerRun asserts a run sends exactly its cap when more
// events are reported, even with several workers racing, and then stops.
func TestTransport_CapsEventsPerRun(t *testing.T) {
	const maxEvents = 20
	fp := newFakePoster(nil)
	// The queue holds every event, so none is dropped before the cap applies.
	tr := newTransportWithCap(fp, context.Background(), defaultWorkers, 2*maxEvents, time.Minute, maxEvents)
	defer tr.Close()

	for i := 0; i < maxEvents+10; i++ {
		tr.Report(sampleUsage())
	}
	waitSignals(t, fp.entered, maxEvents, 2*time.Second)
	waitFor(t, tr.stopped.Load, 2*time.Second)
	select {
	case <-fp.entered:
		t.Fatalf("sent more than the cap of %d (calls=%d)", maxEvents, fp.calls.Load())
	case <-time.After(100 * time.Millisecond):
	}
}

// TestTransport_CapCountsEveryAttemptEndToEnd drives the real SDK poster: the cap
// bounds requests, not successes, so sends that fail without stopping the run
// still use it up. With one event more than the cap reported, exactly the cap
// reaches the endpoint.
func TestTransport_CapCountsEveryAttemptEndToEnd(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if hits.Add(1)%2 == 1 {
			w.WriteHeader(http.StatusBadRequest) // a failure, never two in a row
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// The alternating responses never fail twice in a row, so only the cap can
	// stop the run; a cap unequal to maxConsecutiveFailures also keeps a failure
	// stop from passing for a cap stop.
	const maxEvents = maxConsecutiveFailures + 2
	tr := newTransportWithCap(NewSDKPoster(srv.URL, srv.Client(), "ua", nil), context.Background(), 1, 16, time.Minute, maxEvents)
	defer tr.Close()

	for i := 0; i < maxEvents+1; i++ {
		tr.Report(sampleUsage())
	}
	waitFor(t, tr.stopped.Load, 2*time.Second)
	time.Sleep(50 * time.Millisecond) // a send after the stop would land here
	if got := hits.Load(); got != maxEvents {
		t.Fatalf("endpoint received %d requests, want exactly %d", got, maxEvents)
	}
}

// TestTransport_BoundsRequestsUnderDefaultPool asserts the per-run request bounds
// with the production pool and Terraform's default ten concurrent callers: a
// backend answering 503 receives at most one send per worker, and one that
// always fails or hangs at most maxConsecutiveFailures plus the other workers'
// in-flight sends.
func TestTransport_BoundsRequestsUnderDefaultPool(t *testing.T) {
	cases := map[string]struct {
		post     func(ctx context.Context) error
		min, max int64
	}{
		"503":    {func(context.Context) error { return &statusError{code: http.StatusServiceUnavailable} }, 1, defaultWorkers},
		"failed": {func(context.Context) error { return errors.New("connection refused") }, maxConsecutiveFailures, maxConsecutiveFailures + defaultWorkers - 1},
		"hung":   {func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }, maxConsecutiveFailures, maxConsecutiveFailures + defaultWorkers - 1},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fp := newFakePoster(func(ctx context.Context, _ Usage) error { return tc.post(ctx) })
			tr := newTransport(fp, context.Background(), defaultWorkers, defaultQueueDepth, 20*time.Millisecond)
			defer tr.Close()

			var wg sync.WaitGroup
			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for j := 0; j < 20; j++ {
						tr.Report(sampleUsage())
						time.Sleep(time.Millisecond)
					}
				}()
			}
			wg.Wait()
			waitFor(t, tr.stopped.Load, 2*time.Second)
			time.Sleep(100 * time.Millisecond) // a send starting after the stop would land here
			if got := fp.calls.Load(); got < tc.min || got > tc.max {
				t.Fatalf("backend received %d sends, want between %d and %d", got, tc.min, tc.max)
			}
		})
	}
}

// syncBuffer collects log output; the workers write to it concurrently.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

// count returns how many captured log entries have level and a message
// containing msg.
func (s *syncBuffer) count(t *testing.T, level, msg string) int {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := tflogtest.MultilineJSONDecode(bytes.NewReader(s.buf.Bytes()))
	if err != nil {
		t.Fatalf("decoding log output: %v", err)
	}
	n := 0
	for _, e := range entries {
		if m, _ := e["@message"].(string); e["@level"] == level && strings.Contains(m, msg) {
			n++
		}
	}
	return n
}

// TestTransport_LogsStopReasonOnce asserts the reason a run stops is logged
// exactly once however many events follow: the cap at WARN (TFCA-B10), a
// failure stop at DEBUG.
func TestTransport_LogsStopReasonOnce(t *testing.T) {
	cases := []struct {
		name      string
		poster    *fakePoster
		maxEvents int64
		level     string
		msg       string
	}{
		{"cap", newFakePoster(nil), 2, "warn", "event cap reached"},
		// The default cap keeps the cap from stopping the run before the 503 does.
		{"failure", newFakePoster(func(context.Context, Usage) error {
			return &statusError{code: http.StatusServiceUnavailable}
		}), maxEventsPerRun, "debug", "stopped client-analytics reporting"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out syncBuffer
			ctx := tflogtest.RootLogger(context.Background(), &out)
			tr := newTransportWithCap(tc.poster, ctx, defaultWorkers, 16, time.Minute, tc.maxEvents)
			defer tr.Close()

			for i := 0; i < 10; i++ {
				tr.Report(sampleUsage())
			}
			waitFor(t, func() bool { return out.count(t, tc.level, tc.msg) > 0 }, 2*time.Second)
			// Give a duplicate from another worker time to appear.
			time.Sleep(50 * time.Millisecond)
			if n := out.count(t, tc.level, tc.msg); n != 1 {
				t.Fatalf("logged the stop reason %d times at %s, want once", n, tc.level)
			}
		})
	}
}

// TestTransport_StopReportsFirstCallerOnly asserts stop reports true to exactly
// one of many concurrent callers, which is what keeps the stop reason logged once
// when several workers hit a stop condition together.
func TestTransport_StopReportsFirstCallerOnly(t *testing.T) {
	tr := newTransport(newFakePoster(nil), context.Background(), 0, 1, time.Minute)
	defer tr.Close()

	var wg sync.WaitGroup
	var firsts atomic.Int64
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tr.stop() {
				firsts.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := firsts.Load(); got != 1 {
		t.Fatalf("stop reported true to %d callers, want exactly 1", got)
	}
	if !tr.stopped.Load() {
		t.Fatal("stop did not mark the run stopped")
	}
}

// TestTransport_StopsOnOverloadEndToEnd drives the real SDK poster against an
// endpoint answering 503: the run stops after exactly one request.
func TestTransport_StopsOnOverloadEndToEnd(t *testing.T) {
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	// One worker: the run stops before the next queued event is taken, so the
	// count is final once the run stops.
	tr := newTransport(NewSDKPoster(srv.URL, srv.Client(), "ua", nil), context.Background(), 1, 8, time.Minute)
	defer tr.Close()

	for i := 0; i < 5; i++ {
		tr.Report(sampleUsage())
	}
	waitFor(t, tr.stopped.Load, 2*time.Second)
	time.Sleep(50 * time.Millisecond) // a send after the stop would land here
	if got := hits.Load(); got != 1 {
		t.Fatalf("endpoint received %d requests, want 1", got)
	}
}

// TestNewTransport_UsesDefaults pins the default pool size, queue depth,
// per-report timeout, and run limits the constructor uses.
func TestNewTransport_UsesDefaults(t *testing.T) {
	tr := NewTransport(newFakePoster(nil), context.Background())
	defer tr.Close()

	if got := cap(tr.queue); got != defaultQueueDepth {
		t.Errorf("queue depth = %d, want %d", got, defaultQueueDepth)
	}
	if tr.timeout != defaultPerReportTimeout {
		t.Errorf("per-report timeout = %s, want %s", tr.timeout, defaultPerReportTimeout)
	}
	if tr.maxEvents != maxEventsPerRun {
		t.Errorf("per-run cap = %d, want %d", tr.maxEvents, maxEventsPerRun)
	}
	if maxEventsPerRun != 10000 {
		t.Errorf("maxEventsPerRun = %d, want 10000", maxEventsPerRun)
	}
	if maxConsecutiveFailures != 5 {
		t.Errorf("maxConsecutiveFailures = %d, want 5", maxConsecutiveFailures)
	}
	if maxConsecutiveFailures <= defaultWorkers {
		t.Errorf("maxConsecutiveFailures = %d, want more than defaultWorkers (%d) so one burst of in-flight failures cannot stop the run", maxConsecutiveFailures, defaultWorkers)
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
