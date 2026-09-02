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
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// panicReporter panics on Report, simulating a fault in the report/enqueue path.
type panicReporter struct{}

func (panicReporter) Report(telemetry.Usage) { panic("reporter boom") }

// TestWrapper_RecoversCrudPanic asserts a CRUD panic becomes error diagnostics
// (no crash) and reports one crash event with Error=true and a stack trace.
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

// TestWrapper_RecoversPanicInReportPath asserts a panic in the reporter after a
// successful call is contained, preserving the successful result.
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
// independent: when both the wrapped call and the reporter panic, both are
// contained and the call still returns clean error diagnostics.
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

// TestWrapper_PanicSkipsChangedAttributeCollection asserts the crash path collects
// no changed attributes even when a real, change-bearing ResourceData is supplied
// to a panicking Create, so it never reflects over a possibly-torn ResourceData.
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

// TestReportSafely_ContainsPanics asserts reportSafely contains a panic from
// either building the payload or reporting it.
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
// normal return from a panic, including panic(nil) (still a panic on Go 1.21+).
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

// TestWrapper_PanicErrorSurfacesStack asserts the operator-visible error from a
// recovered panic carries the panic value and the trimmed stack, on both the CRUD
// diagnostics detail and the import error.
func TestWrapper_PanicErrorSurfacesStack(t *testing.T) {
	r := newTestResource()
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		panic("kaboom-create")
	}
	r.Importer.StateContext = func(context.Context, *schema.ResourceData, interface{}) ([]*schema.ResourceData, error) {
		panic("kaboom-import")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(&recordingReporter{}))

	diags := r.CreateContext(context.Background(), nil, nil)
	if !diags.HasError() {
		t.Fatalf("expected error diagnostics from a recovered panic, got %+v", diags)
	}
	detail := diags[0].Detail
	if !strings.Contains(detail, "kaboom-create") {
		t.Errorf("CRUD panic Detail should include the panic value, got %q", detail)
	}
	if !strings.Contains(detail, ".go:") {
		t.Errorf("CRUD panic Detail should include a stack frame (file:line), got %q", detail)
	}
	if strings.Contains(detail, "/Users/") {
		t.Errorf("CRUD panic Detail leaked an absolute local path: %q", detail)
	}

	_, err := r.Importer.StateContext(context.Background(), nil, nil)
	if err == nil {
		t.Fatal("expected an error from a recovered import panic")
	}
	if !strings.Contains(err.Error(), "kaboom-import") || !strings.Contains(err.Error(), ".go:") {
		t.Errorf("import panic error should include the panic value and a stack frame, got %v", err)
	}
}

// TestWrapper_HappyPathHasNoStackFrames asserts a successful operation emits a
// Usage with no stack frames, so the panic-only path never runs on success.
func TestWrapper_HappyPathHasNoStackFrames(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource() // all CRUD/import entry points are successful no-ops
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

	_ = r.CreateContext(context.Background(), nil, nil)
	_, _ = r.Importer.StateContext(context.Background(), nil, nil)

	for _, u := range rec.snapshot() {
		if u.StackFrames != nil {
			t.Errorf("op %s: happy path must not carry stack frames, got %#v", u.Operation, u.StackFrames)
		}
		if u.Error {
			t.Errorf("op %s: happy path must not be flagged as an error", u.Operation)
		}
	}
}

// TestShortenSourcePath asserts the redaction contract: an absolute path is reduced
// to its last two segments (stripping the username), a short path is untouched, and
// a backslash path is redacted the same way.
func TestShortenSourcePath(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"/Users/cqin/go/src/x/internal/provider/resource_x.go:123", "provider/resource_x.go:123"},
		{"/home/jenkins/go/pkg/mod/github.com/foo/bar/baz.go:9", "bar/baz.go:9"},
		{"provider/resource_x.go:1", "provider/resource_x.go:1"},
		{"file.go:1", "file.go:1"},
		{`C:\Users\jane\proj\pkg\main.go:10`, "pkg/main.go:10"}, // fail closed: username stripped
	} {
		got := shortenSourcePath(tc.in)
		if got != tc.want {
			t.Errorf("shortenSourcePath(%q) = %q, want %q", tc.in, got, tc.want)
		}
		// Structural, separator-agnostic guarantee: at most two segments survive,
		// so no deep path (and thus no username directory) can leak.
		if strings.Count(got, "/") > 1 {
			t.Errorf("shortenSourcePath(%q) = %q kept >2 segments; redaction did not shorten", tc.in, got)
		}
		if strings.Contains(got, "Users") || strings.Contains(got, "jane") || strings.Contains(got, "cqin") {
			t.Errorf("shortenSourcePath(%q) = %q leaked a user path segment", tc.in, got)
		}
	}
}

// TestPanicDetail_CapsFramesForOperator asserts the operator-facing error shows at
// most maxDetailFrames frames (with a truncation marker) even though the full stack
// is still reported to telemetry. It keeps the human-readable error short while the
// panic origin, near the top of the stack, stays visible.
func TestPanicDetail_CapsFramesForOperator(t *testing.T) {
	stack := make([]string, maxStackFrames) // deeper than the detail cap
	for i := range stack {
		stack[i] = fmt.Sprintf("pkg/file.go:%d", i)
	}
	detail := panicDetail("boom", stack)

	if !strings.Contains(detail, "boom") {
		t.Errorf("detail should include the panic value, got %q", detail)
	}
	// The top maxDetailFrames frames are shown; the next one is not.
	if !strings.Contains(detail, fmt.Sprintf("pkg/file.go:%d", maxDetailFrames-1)) {
		t.Errorf("detail should include the top %d frames, got %q", maxDetailFrames, detail)
	}
	if strings.Contains(detail, fmt.Sprintf("pkg/file.go:%d", maxDetailFrames)) {
		t.Errorf("detail should not include frames beyond the top %d, got %q", maxDetailFrames, detail)
	}
	// A truncation marker names how many of how many frames are shown.
	if !strings.Contains(detail, fmt.Sprintf("showing top %d of %d frames", maxDetailFrames, len(stack))) {
		t.Errorf("detail should carry a truncation marker, got %q", detail)
	}
	// The detail body carries exactly the capped number of frame lines.
	frames := 0
	for _, ln := range strings.Split(detail, "\n") {
		if strings.Contains(ln, "pkg/file.go:") {
			frames++
		}
	}
	if frames != maxDetailFrames {
		t.Errorf("detail shows %d frames, want %d", frames, maxDetailFrames)
	}
}

// TestPanicDetail_ShortStackNotTruncated asserts a stack at or under the cap is
// shown in full, with no truncation marker.
func TestPanicDetail_ShortStackNotTruncated(t *testing.T) {
	stack := []string{"pkg/a.go:1", "pkg/b.go:2"}
	detail := panicDetail("boom", stack)
	if strings.Contains(detail, "showing top") {
		t.Errorf("a stack under the cap should not be truncated, got %q", detail)
	}
	for _, f := range stack {
		if !strings.Contains(detail, f) {
			t.Errorf("detail should include frame %q, got %q", f, detail)
		}
	}
}
