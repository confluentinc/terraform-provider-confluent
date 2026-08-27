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
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// This file is the client-analytics transport (TFCA-B5): a small, fixed worker
// pool that delivers Usage events without ever blocking or failing the CRUD
// goroutine that produced them.
//
// The contract with the wrapper (TFCA-B3) is deliberate: the wrapper builds the
// Usage synchronously — while the *schema.ResourceData is still valid — and
// hands the finished value to Report. Report only enqueues; the network send
// happens on a worker. So the decision of what to report is synchronous, but
// the send is not, and a slow or unreachable backend can never delay an apply.
//
// Loss is expected and accepted. Terraform's plugin protocol has no drain hook:
// the provider subprocess is killed the moment the graph walk completes, with no
// chance to flush. A deep queue would not reduce loss, only change which events
// are dropped, so the queue is intentionally shallow and a full queue drops
// immediately — the same log-and-drop terms as a failed send. There are no
// retries.

const (
	// defaultWorkers is the fixed number of concurrent senders. Fixed rather
	// than one goroutine per call so a slow backend cannot accumulate
	// connections during a large apply.
	defaultWorkers = 4

	// defaultQueueDepth is kept shallow — roughly the worker concurrency — for
	// the reasons in the file header. An event that cannot be handed to the
	// queue immediately is dropped immediately.
	defaultQueueDepth = defaultWorkers * 2

	// defaultPerReportTimeout is the hard per-report deadline, mirroring the
	// CLI's 5-second usage-report timeout. It bounds how long a worker may spend
	// on a single send; it does not affect the CRUD goroutine, which never waits
	// on a send.
	defaultPerReportTimeout = 5 * time.Second
)

// Poster delivers a single Usage. The generated terraform-usage/v1 client backs
// the production implementation (see sdkPoster); tests substitute a fake. Post
// must honor ctx cancellation/deadline so the transport's per-report timeout can
// bound a hung backend.
type Poster interface {
	Post(ctx context.Context, u Usage) error
}

// Transport is the bounded-worker delivery mechanism. Construct it once per
// provider process (during provider configuration) and share it across the
// concurrent CRUD goroutines; Report is safe for concurrent use.
//
// It satisfies the reporter seam the wrapper expects (a Report(Usage) method),
// so a *Transport can be installed as the wrapper's reporter directly.
type Transport struct {
	poster  Poster
	queue   chan Usage
	timeout time.Duration
	// logCtx carries the provider's configured logger for the workers, which run
	// after the configuring call has returned. Its cancellation is stripped so it
	// stays usable for the life of the process without leaking a request scope.
	logCtx    context.Context
	done      chan struct{}
	closeOnce sync.Once
}

// NewTransport starts a Transport with the default worker pool, queue depth, and
// per-report timeout. logCtx supplies the logger the workers log dropped events
// through; a nil logCtx falls back to a background context.
func NewTransport(poster Poster, logCtx context.Context) *Transport {
	return newTransport(poster, logCtx, defaultWorkers, defaultQueueDepth, defaultPerReportTimeout)
}

func newTransport(poster Poster, logCtx context.Context, workers, queueDepth int, timeout time.Duration) *Transport {
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

// Report hands one Usage to the worker pool. It never blocks: if every worker is
// busy and the queue is full, the event is dropped immediately (logged at debug
// level), because there is no drain window in which a queued event would fare
// better than a dropped one. This is the method the CRUD wrapper calls, and it
// must stay non-blocking so telemetry can never slow an apply.
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
	ctx, cancel := context.WithTimeout(t.logCtx, t.timeout)
	defer cancel()
	if err := t.poster.Post(ctx, u); err != nil {
		// Log and drop. Never retry, and never surface as a CRUD error.
		tflog.Debug(t.logCtx, "dropped client-analytics event: report failed", map[string]interface{}{
			"resource_type": u.ResourceType,
			"operation":     string(u.Operation),
			"error":         err.Error(),
		})
	}
}

// Close stops the workers. Production never calls it — the provider subprocess
// is killed at the end of the graph walk — but tests use it to end the workers
// deterministically. It must not race with Report; after Close, Report must not
// be called. Close is idempotent.
func (t *Transport) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
	})
}
