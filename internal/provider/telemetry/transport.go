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
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Bounded-worker transport for client-analytics events. Report enqueues an event
// and returns; a fixed pool of workers sends it in the background, so a slow or
// unreachable backend never delays the caller. Delivery is best-effort: a full
// queue and a failed send are both logged and dropped, with no retries.
//
// A run (one provider process) stops reporting for good once the backend answers
// 429 or 5xx, once maxConsecutiveFailures sends in a row have failed, or once it
// has sent maxEventsPerRun events. This bounds what one run can send to a
// struggling backend, and lets the server shed load by rejecting requests.

const (
	// defaultWorkers is the fixed number of concurrent senders, so a slow backend
	// cannot accumulate one connection per call during a large apply.
	defaultWorkers = 4

	// defaultQueueDepth is kept shallow; an event that cannot be queued
	// immediately is dropped immediately.
	defaultQueueDepth = defaultWorkers * 2

	// defaultPerReportTimeout bounds how long a worker may spend on a single send.
	defaultPerReportTimeout = 5 * time.Second

	// maxEventsPerRun caps how many events one run sends (TFCA-B10). It is a
	// backstop against an unusually large or runaway run, set well above an
	// ordinary run's one event per managed resource per refresh; it is not a
	// sampling policy.
	maxEventsPerRun = 10000

	// maxConsecutiveFailures is how many sends in a row may fail before the run
	// stops reporting; a 429 or 5xx response stops it at once instead. It exceeds
	// defaultWorkers so one burst of concurrent in-flight failures, such as a
	// brief network blip, cannot stop the run on its own. A backend that always
	// fails therefore sees at most maxConsecutiveFailures+defaultWorkers-1 sends
	// per run.
	maxConsecutiveFailures = defaultWorkers + 1
)

// Poster delivers a single Usage. Post must honor ctx so the per-report timeout
// can bound a hung backend.
type Poster interface {
	Post(ctx context.Context, u Usage) error
}

// noopPoster replaces a nil Poster so a misconfigured transport drops events
// instead of nil-panicking a worker goroutine and crashing the process.
type noopPoster struct{}

func (noopPoster) Post(context.Context, Usage) error { return nil }

// Transport is the bounded-worker delivery mechanism. Construct it once and share
// it across concurrent callers; Report is safe for concurrent use.
type Transport struct {
	poster  Poster
	queue   chan Usage
	timeout time.Duration
	// maxEvents is the per-run cap on sends (maxEventsPerRun outside tests).
	maxEvents int64
	// logCtx carries the workers' logger; its cancellation is stripped so it
	// stays usable for the life of the process.
	logCtx    context.Context
	done      chan struct{}
	closeOnce sync.Once

	// stopped is set once reporting stops for the rest of the run.
	stopped atomic.Bool
	// sent counts events the workers have taken to send, against maxEvents. It
	// can pass maxEvents by up to the number of workers once the cap is reached;
	// at most maxEvents are ever sent.
	sent atomic.Int64
	// failures counts consecutive failed sends; a success resets it.
	failures atomic.Int64
}

// NewTransport starts a Transport with the default pool size, queue depth,
// per-report timeout, and per-run cap. A nil logCtx falls back to a background
// context.
func NewTransport(poster Poster, logCtx context.Context) *Transport {
	return newTransport(poster, logCtx, defaultWorkers, defaultQueueDepth, defaultPerReportTimeout)
}

func newTransport(poster Poster, logCtx context.Context, workers, queueDepth int, timeout time.Duration) *Transport {
	return newTransportWithCap(poster, logCtx, workers, queueDepth, timeout, maxEventsPerRun)
}

// newTransportWithCap is newTransport with an explicit per-run cap, so tests can
// exercise the cap without sending maxEventsPerRun events.
func newTransportWithCap(poster Poster, logCtx context.Context, workers, queueDepth int, timeout time.Duration, maxEvents int64) *Transport {
	if poster == nil {
		poster = noopPoster{}
	}
	if logCtx == nil {
		logCtx = context.Background()
	}
	t := &Transport{
		poster:    poster,
		queue:     make(chan Usage, queueDepth),
		timeout:   timeout,
		maxEvents: maxEvents,
		logCtx:    context.WithoutCancel(logCtx),
		done:      make(chan struct{}),
	}
	for i := 0; i < workers; i++ {
		go t.worker()
	}
	return t
}

// Report hands one Usage to the worker pool. It never blocks: when the queue is
// full the event is dropped and logged, so telemetry can never slow the caller.
// Once reporting has stopped for the run, the event is dropped without being
// queued.
func (t *Transport) Report(u Usage) {
	if t.stopped.Load() {
		return
	}
	select {
	case t.queue <- u:
	default:
		tflog.Debug(t.logCtx, "dropped client-analytics event: transport queue full", map[string]interface{}{
			"resource_type": u.ResourceType,
			"operation":     string(u.Operation),
		})
	}
}

func (t *Transport) worker() {
	for {
		select {
		case <-t.done:
			return
		case u, ok := <-t.queue:
			if !ok {
				return
			}
			t.deliver(u)
		}
	}
}

func (t *Transport) deliver(u Usage) {
	// Drop an event that was queued before reporting stopped.
	if t.stopped.Load() {
		return
	}
	// Count the send here, the only place one is attempted, so the cap bounds
	// sends exactly regardless of what the queue dropped.
	if t.sent.Add(1) > t.maxEvents {
		if t.stop() {
			tflog.Warn(t.logCtx, "client-analytics event cap reached: not reporting further events in this run", map[string]interface{}{
				"max_events": t.maxEvents,
			})
		}
		return
	}
	// A panic from Post runs on this worker goroutine, outside the caller's
	// recover; catch it so a bad send can never crash the process.
	defer func() {
		if r := recover(); r != nil {
			tflog.Debug(t.logCtx, "dropped client-analytics event: report panicked", map[string]interface{}{
				"resource_type": u.ResourceType,
				"operation":     string(u.Operation),
				"panic":         fmt.Sprint(r),
			})
			t.recordFailure(fmt.Errorf("report panicked: %v", r))
		}
	}()
	ctx, cancel := context.WithTimeout(t.logCtx, t.timeout)
	defer cancel()
	if err := t.poster.Post(ctx, u); err != nil {
		// Log and drop; never retry.
		tflog.Debug(t.logCtx, "dropped client-analytics event: report failed", map[string]interface{}{
			"resource_type": u.ResourceType,
			"operation":     string(u.Operation),
			"error":         err.Error(),
		})
		t.recordFailure(err)
		return
	}
	t.failures.Store(0)
}

// recordFailure counts a failed send and stops reporting for the rest of the
// run when the backend signals overload (429 or 5xx) or after
// maxConsecutiveFailures sends in a row have failed.
func (t *Transport) recordFailure(err error) {
	failures := t.failures.Add(1)
	var se *statusError
	overloaded := errors.As(err, &se) && (se.code == http.StatusTooManyRequests || (se.code >= http.StatusInternalServerError && se.code < 600))
	if (overloaded || failures >= maxConsecutiveFailures) && t.stop() {
		tflog.Debug(t.logCtx, "stopped client-analytics reporting for the rest of this run", map[string]interface{}{
			"consecutive_failures": failures,
			"error":                err.Error(),
		})
	}
}

// stop ends reporting for the rest of the run. It reports whether this call
// stopped it, so the reason is logged exactly once.
func (t *Transport) stop() bool {
	return t.stopped.CompareAndSwap(false, true)
}

// Close signals the workers to stop. Only tests call it (production lets the
// subprocess exit); it does not drain queued events and is idempotent.
func (t *Transport) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}
