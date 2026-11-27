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
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Bounded-worker transport for client-analytics events. Report enqueues an event
// and returns; a fixed pool of workers sends it in the background, so a slow or
// unreachable backend never delays the caller. Delivery is best-effort: a full
// queue and a failed send are both logged and dropped, with no retries. At
// process exit, Drain gives unsent events a bounded wait.

const (
	// defaultWorkers is the fixed number of concurrent senders, so a slow backend
	// cannot accumulate one connection per call during a large apply.
	defaultWorkers = 4

	// defaultQueueDepth is kept shallow; an event that cannot be queued
	// immediately is dropped immediately.
	defaultQueueDepth = defaultWorkers * 2

	// defaultPerReportTimeout bounds how long a worker may spend on a single send.
	defaultPerReportTimeout = 5 * time.Second
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
	// logCtx carries the workers' logger; its cancellation is stripped so it
	// stays usable for the life of the process.
	logCtx    context.Context
	done      chan struct{}
	closeOnce sync.Once

	// mu guards the fields below, which let Drain wait for unsent events.
	mu sync.Mutex
	// pending counts events that are queued or being sent.
	pending int
	// idle is closed whenever pending is zero.
	idle chan struct{}
	// activeAt is when pending last rose from zero or an event last settled.
	activeAt time.Time
	// lastFailed reports whether the most recently settled send failed.
	lastFailed bool
}

// NewTransport starts a Transport with the default pool size, queue depth, and
// per-report timeout. A nil logCtx falls back to a background context.
func NewTransport(poster Poster, logCtx context.Context) *Transport {
	return newTransport(poster, logCtx, defaultWorkers, defaultQueueDepth, defaultPerReportTimeout)
}

func newTransport(poster Poster, logCtx context.Context, workers, queueDepth int, timeout time.Duration) *Transport {
	if poster == nil {
		poster = noopPoster{}
	}
	if logCtx == nil {
		logCtx = context.Background()
	}
	t := &Transport{
		poster:  poster,
		queue:   make(chan Usage, queueDepth),
		timeout: timeout,
		logCtx:  context.WithoutCancel(logCtx),
		done:    make(chan struct{}),
		idle:    make(chan struct{}),
	}
	close(t.idle)
	for i := 0; i < workers; i++ {
		go t.worker()
	}
	return t
}

// Report hands one Usage to the worker pool. It never blocks: when the queue is
// full the event is dropped and logged, so telemetry can never slow the caller.
func (t *Transport) Report(u Usage) {
	if !t.enqueue(u) {
		tflog.Debug(t.logCtx, "dropped client-analytics event: transport queue full", map[string]interface{}{
			"resource_type": u.ResourceType,
			"operation":     string(u.Operation),
		})
	}
}

// enqueue queues u without blocking and reports whether it was queued. It holds
// mu across the send so a queued event is always counted as pending.
func (t *Transport) enqueue(u Usage) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case t.queue <- u:
		if t.pending == 0 {
			t.idle = make(chan struct{})
			t.activeAt = time.Now()
		}
		t.pending++
		return true
	default:
		return false
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
			t.settle(t.deliver(u))
		}
	}
}

// settle marks one pending event as sent or dropped.
func (t *Transport) settle(sent bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pending--
	t.activeAt = time.Now()
	t.lastFailed = !sent
	if t.pending == 0 {
		close(t.idle)
	}
}

// deliver sends u and reports whether it was sent; a panic counts as not sent.
func (t *Transport) deliver(u Usage) (sent bool) {
	// A panic from Post runs on this worker goroutine, outside the caller's
	// recover; catch it so a bad send can never crash the process.
	defer func() {
		if r := recover(); r != nil {
			tflog.Debug(t.logCtx, "dropped client-analytics event: report panicked", map[string]interface{}{
				"resource_type": u.ResourceType,
				"operation":     string(u.Operation),
				"panic":         fmt.Sprint(r),
			})
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
		return false
	}
	return true
}

// Drain waits until no event is queued or being sent, and reports whether it got
// there. It waits at most until timeout after the transport was last active
// before the call (pending rose from zero or an event settled), and not at all
// when the most recently settled send failed, so a failing or stuck backend is
// not waited on. It is for process exit: it does not stop the workers, and a
// send still running when it returns is abandoned.
func (t *Transport) Drain(timeout time.Duration) bool {
	t.mu.Lock()
	idle, pending, activeAt, lastFailed := t.idle, t.pending, t.activeAt, t.lastFailed
	t.mu.Unlock()
	if pending == 0 {
		return true
	}
	wait := time.Until(activeAt.Add(timeout))
	if lastFailed || wait <= 0 {
		return false
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-idle:
		return true
	case <-timer.C:
		return false
	}
}

// Close signals the workers to stop. Only tests call it (production lets the
// subprocess exit); it does not drain queued events and is idempotent.
func (t *Transport) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}
