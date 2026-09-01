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
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// panicReporter panics on Report, simulating a fault in the report/enqueue path.
type panicReporter struct{}

func (panicReporter) Report(telemetry.Usage) { panic("reporter boom") }

// TestWrapper_RecoversCrudPanic asserts a panic in a wrapped CRUD call is
// converted to error diagnostics (not propagated), the process does not crash,
// and exactly one crash event is reported with Error=true, no changed
// attributes, and a stack trace.
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
		{telemetry.OperationRead,
			func(r *schema.Resource, fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics) {
				r.ReadContext = fn
			},
			func(r *schema.Resource) diag.Diagnostics { return r.ReadContext(context.Background(), nil, nil) }},
		{telemetry.OperationUpdate,
			func(r *schema.Resource, fn func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics) {
				r.UpdateContext = fn
			},
			func(r *schema.Resource) diag.Diagnostics { return r.UpdateContext(context.Background(), nil, nil) }},
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
			// On the panic path we must not reflect over a possibly-torn
			// ResourceData, so no changed attributes are collected.
			if u.ChangedAttributes == nil || len(u.ChangedAttributes) != 0 {
				t.Errorf("crash event ChangedAttributes should be empty non-nil, got %#v", u.ChangedAttributes)
			}
		})
	}
}

// TestWrapper_RecoversImportPanic asserts the import path recovers a panic,
// returns an error with no partial result, and reports an IMPORT crash event.
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
	if rec.count() != 1 {
		t.Fatalf("expected exactly 1 crash event, got %d", rec.count())
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

// TestWrapper_StackFramesAreCapped asserts a deep stack is truncated to
// maxStackFrames, so a runaway recursion can't bloat a crash payload.
func TestWrapper_StackFramesAreCapped(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	var recurse func(int)
	recurse = func(n int) {
		if n == 0 {
			panic("deep kaboom")
		}
		recurse(n - 1)
	}
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		recurse(maxStackFrames * 4) // far deeper than the cap
		return nil
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))
	_ = r.CreateContext(context.Background(), nil, nil)

	frames := rec.snapshot()[0].StackFrames
	if len(frames) == 0 {
		t.Fatal("expected stack frames")
	}
	if len(frames) > maxStackFrames {
		t.Errorf("stack has %d frames, want <= %d (cap not applied)", len(frames), maxStackFrames)
	}
}

// TestWrapper_RecoversPanicInReportPath asserts a panic while reporting (here a
// panicking reporter) after a *successful* CRUD call is contained and never
// reaches the caller, and the underlying successful result is preserved.
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

// TestWrapper_CrashReportPanicIsContained asserts the two recovers are
// independent: when the wrapped call panics AND the reporter also panics while
// reporting the crash, both are contained — the call still returns clean error
// diagnostics and nothing escapes. This is the "endpoint stubbed to fail at
// panic time" case; the bounded-timeout behavior for a *hung* endpoint is the
// reporter's own (TFCA-B5's transport), which replaces the no-op reporter used
// on this path today.
func TestWrapper_CrashReportPanicIsContained(t *testing.T) {
	r := newTestResource()
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		panic("kaboom")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(panicReporter{}))

	var diags diag.Diagnostics
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("a panic escaped when both the call and the reporter panicked: %v", p)
			}
		}()
		diags = r.CreateContext(context.Background(), nil, nil)
	}()
	if !diags.HasError() {
		t.Errorf("expected error diagnostics from the recovered CRUD panic, got %+v", diags)
	}
}

// TestWrapper_PanicSkipsChangedAttributeCollection asserts the crash path does
// not reflect over ResourceData: even when a real, change-bearing ResourceData
// is supplied to a panicking Create, the emitted crash event carries no changed
// attributes. This exercises the rec==nil guard specifically — unlike the nil-d
// cases above, changedAttributeNames on this d WOULD return ["config","topic_name"]
// if the guard were removed, so the test fails if the guard regresses.
func TestWrapper_PanicSkipsChangedAttributeCollection(t *testing.T) {
	rec := &recordingReporter{}
	// Schema modeled on confluent_kafka_topic: a TypeMap "config" plus a scalar.
	r := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"config":     {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
			"topic_name": {Type: schema.TypeString, Optional: true},
		},
		CreateContext: func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
			panic("kaboom")
		},
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_kafka_topic": r}, testWrapConfig(rec))

	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"config":     map[string]interface{}{"cleanup.policy": "compact"},
		"topic_name": "orders",
	})

	var diags diag.Diagnostics
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("panic escaped the wrapper: %v", p)
			}
		}()
		diags = r.CreateContext(context.Background(), d, nil)
	}()
	if !diags.HasError() {
		t.Errorf("expected error diagnostics from a recovered panic")
	}
	u := rec.snapshot()[0]
	if u.ChangedAttributes == nil {
		t.Errorf("ChangedAttributes must be non-nil so it serializes as [] not null")
	}
	if len(u.ChangedAttributes) != 0 {
		t.Errorf("crash path must not collect changed attributes from ResourceData, got %#v", u.ChangedAttributes)
	}
}

// TestReportSafely_ContainsPanics directly exercises reportSafely's recover: a
// panic while BUILDING the payload and a panic while REPORTING are both
// contained, so neither can crash the process the wrapper protects. The build
// case is the payload-construction failure mode (e.g. reflecting over a torn
// ResourceData) that the second recover exists for.
func TestReportSafely_ContainsPanics(t *testing.T) {
	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("reportSafely let a payload-build panic escape: %v", p)
			}
		}()
		reportSafely(testWrapConfig(&recordingReporter{}), func() telemetry.Usage {
			panic("build boom")
		})
	}()

	func() {
		defer func() {
			if p := recover(); p != nil {
				t.Fatalf("reportSafely let a reporter panic escape: %v", p)
			}
		}()
		reportSafely(testWrapConfig(panicReporter{}), func() telemetry.Usage {
			return telemetry.Usage{}
		})
	}()
}

// TestRunGuarded_ClassifiesReturnAndPanic asserts runGuarded distinguishes a
// normal return from a panic, including the panic(nil) edge case (Go 1.21+ turns
// it into a *runtime.PanicNilError, so it must still be classified as a panic).
func TestRunGuarded_ClassifiesReturnAndPanic(t *testing.T) {
	if rec, stack := runGuarded(func() {}); rec != nil || stack != nil {
		t.Errorf("normal return: got rec=%v stack=%v, want nil/nil", rec, stack)
	}
	if rec, stack := runGuarded(func() { panic("boom") }); rec == nil || len(stack) == 0 {
		t.Errorf("panic: got rec=%v stack len=%d, want non-nil rec and frames", rec, len(stack))
	}
	if rec, _ := runGuarded(func() { panic(nil) }); rec == nil {
		t.Errorf("panic(nil) must still be classified as a panic (rec != nil)")
	}
}
