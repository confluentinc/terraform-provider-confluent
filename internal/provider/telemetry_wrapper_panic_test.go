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

package provider

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// panicReporter panics on Report, simulating a fault in the enqueue path.
type panicReporter struct{}

func (panicReporter) Report(telemetry.Usage) { panic("reporter boom") }

// hangPoster blocks Post until its context is cancelled, simulating a hung
// backend that is only released by the transport's per-report timeout.
type hangPoster struct{}

func (hangPoster) Post(ctx context.Context, _ telemetry.Usage) error {
	<-ctx.Done()
	return ctx.Err()
}

// TestWrapper_RecoversCrudPanic asserts a panic in a wrapped CRUD call is
// converted to error diagnostics (not propagated), the process does not crash,
// and exactly one crash event is reported with Error=true and a stack trace.
func TestWrapper_RecoversCrudPanic(t *testing.T) {
	for _, tc := range []struct {
		op    telemetry.Operation
		apply func(r *schema.Resource, panicFn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics)
		call  func(r *schema.Resource) diag.Diagnostics
	}{
		{telemetry.OperationCreate,
			func(r *schema.Resource, fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics) {
				r.CreateContext = fn
			},
			func(r *schema.Resource) diag.Diagnostics { return r.CreateContext(context.Background(), nil, nil) }},
		{telemetry.OperationDelete,
			func(r *schema.Resource, fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics) {
				r.DeleteContext = fn
			},
			func(r *schema.Resource) diag.Diagnostics { return r.DeleteContext(context.Background(), nil, nil) }},
	} {
		t.Run(string(tc.op), func(t *testing.T) {
			rec := &recordingReporter{}
			r := newTestResource()
			tc.apply(r, func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
				panic("kaboom")
			})
			wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

			var diags diag.Diagnostics
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Fatalf("panic escaped the wrapper: %v", p)
					}
				}()
				diags = tc.call(r)
			}()

			if !diags.HasError() {
				t.Errorf("expected error diagnostics from a recovered panic, got %+v", diags)
			}
			if rec.count() != 1 {
				t.Fatalf("expected exactly 1 crash event, got %d", rec.count())
			}
			u := rec.snapshot()[0]
			if !u.Error {
				t.Errorf("crash event must have Error=true")
			}
			if u.Operation != tc.op {
				t.Errorf("Operation = %q, want %q", u.Operation, tc.op)
			}
			if len(u.StackFrames) == 0 {
				t.Errorf("crash event must carry stack frames")
			}
		})
	}
}

// TestWrapper_RecoversImportPanic asserts the import path recovers a panic,
// returns an error, and reports an IMPORT crash event.
func TestWrapper_RecoversImportPanic(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	r.Importer.StateContext = func(context.Context, *schema.ResourceData, interface{}) ([]*schema.ResourceData, error) {
		panic("import kaboom")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

	var out []*schema.ResourceData
	var err error
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("panic escaped the import wrapper: %v", p)
			}
		}()
		out, err = r.Importer.StateContext(context.Background(), nil, nil)
	}()

	if err == nil {
		t.Errorf("expected an error from a recovered import panic")
	}
	if out != nil {
		t.Errorf("expected nil imported data on panic, got %v", out)
	}
	u := rec.snapshot()[0]
	if u.Operation != telemetry.OperationImport || !u.Error || len(u.StackFrames) == 0 {
		t.Errorf("expected an IMPORT crash event with a stack, got %+v", u)
	}
}

// TestWrapper_StackFramesAreRedacted asserts the emitted stack carries no
// absolute local paths — only shortened file:line frames.
func TestWrapper_StackFramesAreRedacted(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		panic("kaboom")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))
	_ = r.CreateContext(context.Background(), nil, nil)

	frames := rec.snapshot()[0].StackFrames
	if len(frames) == 0 {
		t.Fatal("expected stack frames")
	}
	for _, f := range frames {
		if strings.HasPrefix(f, "/") {
			t.Errorf("frame %q is an absolute path", f)
		}
		if strings.Contains(f, "/Users/") || strings.Contains(f, "\\Users\\") {
			t.Errorf("frame %q leaks a local filesystem path", f)
		}
		if !strings.Contains(f, ".go:") {
			t.Errorf("frame %q is not a file:line location", f)
		}
	}
}

// TestWrapper_RecoversPanicInReportPath asserts a panic while reporting (here a
// panicking reporter) is contained and never reaches the caller, and the
// underlying successful CRUD result is preserved.
func TestWrapper_RecoversPanicInReportPath(t *testing.T) {
	r := newTestResource() // CreateContext is a successful no-op
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(panicReporter{}))

	var diags diag.Diagnostics
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("a panic in the report path escaped: %v", p)
			}
		}()
		diags = r.CreateContext(context.Background(), nil, nil)
	}()
	if diags.HasError() {
		t.Errorf("a successful CRUD call must not be turned into an error by a reporter panic: %+v", diags)
	}
}

// TestWrapper_CrashReportDoesNotBlockOnHungBackend asserts that with the real
// bounded-worker transport pointed at a hung backend, a forced panic still
// returns clean error diagnostics promptly — the non-blocking enqueue means the
// crash report never delays the return.
func TestWrapper_CrashReportDoesNotBlockOnHungBackend(t *testing.T) {
	transport := telemetry.NewTransport(hangPoster{}, context.Background())
	defer transport.Close()

	r := newTestResource()
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		panic("kaboom")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, telemetryWrapConfig{
		reporter:        transport,
		providerVersion: "9.9.9-test",
	})

	start := time.Now()
	diags := r.CreateContext(context.Background(), nil, nil)
	elapsed := time.Since(start)

	if !diags.HasError() {
		t.Errorf("expected error diagnostics from a recovered panic")
	}
	if elapsed > 2*time.Second {
		t.Fatalf("crash path blocked on the hung backend for %s; enqueue must be non-blocking", elapsed)
	}
}
