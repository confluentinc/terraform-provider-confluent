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
// queue and a failed send are both logged and dropped, with no retries.

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
	}
	for i := 0; i < workers; i++ {
		go t.worker()
	}
	return t
}

// Report hands one Usage to the worker pool. It never blocks: when the queue is
// full the event is dropped and logged, so telemetry can never slow the caller.
func (t *Transport) Report(u Usage) {
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
	}
}

// Close signals the workers to stop. Only tests call it (production lets the
// subprocess exit); it does not drain queued events and is idempotent.
func (t *Transport) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}
