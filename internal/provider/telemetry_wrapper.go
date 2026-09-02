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
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"weak"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// Central client-analytics wrapping. Replaces every managed resource's CRUD and
// import entry points, once, with wrappers that build a telemetry.Usage and hand
// it to a reporter.
//
// The wrappers are also the provider's only panic recovery (TFCA-B4): a panic in
// a wrapped CRUD or import call is recovered and converted to an error result
// (whose detail carries the trimmed stack, so the operator can still diagnose it
// just as they could from the pre-recovery crash), so it no longer crashes the
// provider process, and is reported as a crash event (Error=true, with the same
// trimmed stack trace). Building and reporting the Usage runs
// under its own recover, separate from the one guarding the inner call, so a
// panic while assembling or enqueuing a report — including after a *successful*
// call — is contained too. Reporting stays synchronous and simply hands the Usage
// to the reporter (a no-op today); the bounded-worker transport that makes that
// hand-off non-blocking (TFCA-B5) and the opt-out wiring (TFCA-B6) land
// separately.

// telemetryReporter receives a populated Usage; Report should not block the caller.
type telemetryReporter interface {
	Report(telemetry.Usage)
}

// noopTelemetryReporter drops every Usage.
type noopTelemetryReporter struct{}

func (noopTelemetryReporter) Report(telemetry.Usage) {}

// telemetryWrapConfig carries the process-scoped inputs used to populate a Usage.
type telemetryWrapConfig struct {
	// reporter receives each Usage; a nil reporter is treated as a no-op.
	reporter telemetryReporter

	// providerVersion is the released provider version.
	providerVersion string

	// terraformVersion returns Terraform Core's version. It is a func because Core
	// sets it during ConfigureProvider, after New returns; a nil func yields "".
	terraformVersion func() string
}

func (c telemetryWrapConfig) report(u telemetry.Usage) {
	r := c.reporter
	if r == nil {
		r = noopTelemetryReporter{}
	}
	r.Report(u)
}

func (c telemetryWrapConfig) tfVersion() string {
	if c.terraformVersion == nil {
		return ""
	}
	return c.terraformVersion()
}

// wrappedResources makes wrapping idempotent: a resource wrapped once is skipped
// on later passes. Keys are weak.Pointers with a cleanup, so an entry never keeps
// the resource (or the provider graph its closures capture) alive.
var wrappedResources sync.Map // map[weak.Pointer[schema.Resource]]struct{}

// wrapResourcesMapForTelemetry wraps every resource's CRUD and import entry
// points. Run it after ResourcesMap is built; it is safe to call more than once.
func wrapResourcesMapForTelemetry(resources map[string]*schema.Resource, cfg telemetryWrapConfig) {
	if cfg.reporter == nil {
		cfg.reporter = noopTelemetryReporter{}
	}
	for resourceType, r := range resources {
		if r == nil {
			continue
		}
		key := weak.Make(r)
		if _, alreadyWrapped := wrappedResources.LoadOrStore(key, struct{}{}); alreadyWrapped {
			continue
		}
		// Drop the weak stub once the resource is collected. The cleanup
		// captures only the weak key, never r, so it cannot keep r alive.
		runtime.AddCleanup(r, wrappedResources.Delete, any(key))
		wrapResourceForTelemetry(resourceType, r, cfg)
	}
}

// resourceWrapped reports whether r has been wrapped (used by tests).
func resourceWrapped(r *schema.Resource) bool {
	_, ok := wrappedResources.Load(weak.Make(r))
	return ok
}

// wrapResourceForTelemetry wraps each entry point only when it is already non-nil.
// A nil UpdateContext means the resource has no update; wrapping it would nil-panic
// and make the SDK treat the resource as updatable.
func wrapResourceForTelemetry(resourceType string, r *schema.Resource, cfg telemetryWrapConfig) {
	if r.CreateContext != nil {
		r.CreateContext = wrapContextFunc(resourceType, r.Schema, telemetry.OperationCreate, r.CreateContext, cfg)
	}
	if r.ReadContext != nil {
		r.ReadContext = wrapContextFunc(resourceType, r.Schema, telemetry.OperationRead, r.ReadContext, cfg)
	}
	if r.UpdateContext != nil {
		r.UpdateContext = wrapContextFunc(resourceType, r.Schema, telemetry.OperationUpdate, r.UpdateContext, cfg)
	}
	if r.DeleteContext != nil {
		r.DeleteContext = wrapContextFunc(resourceType, r.Schema, telemetry.OperationDelete, r.DeleteContext, cfg)
	}
	if r.Importer != nil && r.Importer.StateContext != nil {
		r.Importer.StateContext = wrapImportFunc(resourceType, r.Schema, r.Importer.StateContext, cfg)
	}
}

// wrapContextFunc wraps a Create/Read/Update/DeleteContext function: it times the
// inner call and reports a Usage after it returns. Reporting is synchronous because
// the SDK reads ResourceData as soon as the CRUD call returns.
func wrapContextFunc(resourceType string, resourceSchema map[string]*schema.Schema, op telemetry.Operation, inner func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		// Recover a panic from the wrapped CRUD call — the provider's only panic
		// recovery. Without it a panic in any resource crashes the whole provider
		// process. On panic we convert to error diagnostics and still report a
		// crash event, rather than letting it propagate.
		rec, stack := runGuarded(func() { diags = inner(ctx, d, meta) })
		if rec != nil {
			diags = panicDiagnostics(resourceType, op, rec, stack)
		}

		// Build and report the Usage under its own recover (reportSafely): the
		// recover above only guards the inner call, so a panic while reflecting
		// over ResourceData or formatting the payload here would otherwise escape
		// and crash the very process this feature protects.
		reportSafely(cfg, func() telemetry.Usage {
			changed := []string{}
			if rec == nil {
				changed = changedAttributeNames(d, resourceSchema, op)
			}
			return cfg.buildUsage(resourceType, op, seq, timer, changed, rec != nil || diags.HasError(), stack)
		})
		return diags
	}
}

// wrapImportFunc wraps Importer.StateContext, which has a different signature and
// carries no attribute diff, so ChangedAttributes is always the empty slice.
func wrapImportFunc(resourceType string, resourceSchema map[string]*schema.Schema, inner schema.StateContextFunc, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) ([]*schema.ResourceData, error) {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) (imported []*schema.ResourceData, err error) {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		// Recover a panic on the import path on the same terms as the CRUD path:
		// convert it to an error, discard any partial result, and still report a
		// crash event.
		rec, stack := runGuarded(func() { imported, err = inner(ctx, d, meta) })
		if rec != nil {
			imported = nil
			err = panicError(resourceType, telemetry.OperationImport, rec, stack)
		}

		reportSafely(cfg, func() telemetry.Usage {
			return cfg.buildUsage(resourceType, telemetry.OperationImport, seq, timer, []string{}, rec != nil || err != nil, stack)
		})
		return imported, err
	}
}

// buildUsage assembles a Usage from the process-scoped config and the
// per-operation inputs. stack is non-empty only on the panic path, so a normal
// operation omits stack_frames.
func (c telemetryWrapConfig) buildUsage(resourceType string, op telemetry.Operation, seq int64, timer telemetry.Timer, changed []string, errored bool, stack []string) telemetry.Usage {
	return telemetry.Usage{
		RunID:             telemetry.RunID(),
		Sequence:          seq,
		StartedAt:         timer.StartTime(),
		DurationMs:        timer.Elapsed().Milliseconds(),
		OS:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		ProviderVersion:   c.providerVersion,
		TerraformVersion:  c.tfVersion(),
		ResourceType:      resourceType,
		Operation:         op,
		ChangedAttributes: changed,
		Error:             errored,
		StackFrames:       stack,
	}
}

// runGuarded runs fn, recovering any panic. It returns the recovered value (nil
// if fn returned normally) and, on panic, a trimmed stack trace. The stack is
// captured inside the recovering defer, while the panic is still unwinding, so it
// reflects the panicking call site rather than just this recovery point.
func runGuarded(fn func()) (rec interface{}, stack []string) {
	defer func() {
		if r := recover(); r != nil {
			rec = r
			stack = trimmedStackFrames()
		}
	}()
	fn()
	return
}

// reportSafely builds and reports a Usage, recovering (and swallowing) any panic
// from the build or the enqueue. Telemetry must never crash the process it is
// observing, so a failure here is dropped, not propagated.
func reportSafely(cfg telemetryWrapConfig, build func() telemetry.Usage) {
	defer func() { _ = recover() }()
	cfg.report(build())
}

// panicDiagnostics converts a recovered panic into error diagnostics returned in
// place of the wrapped CRUD call's result. The stack is included in the Detail
// (see panicDetail) so the operator keeps a diagnosable trace.
func panicDiagnostics(resourceType string, op telemetry.Operation, rec interface{}, stack []string) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("the Confluent provider recovered from a panic during %s of %s", strings.ToLower(string(op)), resourceType),
		Detail:   panicDetail(rec, stack),
	}}
}

// panicError is the import-path equivalent of panicDiagnostics.
func panicError(resourceType string, op telemetry.Operation, rec interface{}, stack []string) error {
	return fmt.Errorf("the Confluent provider recovered from a panic during %s of %s: %s", strings.ToLower(string(op)), resourceType, panicDetail(rec, stack))
}

// panicDetail renders the recovered panic value with its trimmed stack for the
// operator-visible error. Before this wrapper existed, an uncaught panic crashed
// the provider and Terraform Core printed the full stack to the user; surfacing
// the (already path-redacted) trimmed stack here keeps a recovered panic just as
// diagnosable — and just as present in a bug report — without the crash, and
// without relying on the telemetry reporter (a no-op until TFCA-B5).
func panicDetail(rec interface{}, stack []string) string {
	if len(stack) == 0 {
		return fmt.Sprintf("%v", rec)
	}
	return fmt.Sprintf("%v\n\n%s", rec, strings.Join(stack, "\n"))
}

// maxStackFrames caps how many frames a crash payload carries, so a deep stack
// can't bloat a report.
const maxStackFrames = 50

// trimmedStackFrames captures the current goroutine's stack and reduces it to
// file:line frames with local absolute paths stripped, mirroring the CLI's
// panic-report redaction. It keeps only lines that reference a .go source
// location and shortens each to its last two path segments, so no user's
// absolute filesystem path (e.g. /Users/<name>/...) reaches the wire.
func trimmedStackFrames() []string {
	lines := strings.Split(string(debug.Stack()), "\n")
	frames := make([]string, 0, len(lines))
	for _, line := range lines {
		loc := strings.TrimSpace(line)
		if !strings.Contains(loc, ".go:") {
			continue
		}
		// A stack location line looks like "/abs/path/pkg/file.go:123 +0x1f".
		// Drop the trailing " +0x.." program-counter offset specifically, rather
		// than cutting at the first space, so a build path containing a space
		// (e.g. "/Users/Jane Doe/...") is not truncated mid-path.
		if off := strings.LastIndex(loc, " +0x"); off >= 0 {
			loc = loc[:off]
		}
		frames = append(frames, shortenSourcePath(loc))
		if len(frames) >= maxStackFrames {
			break
		}
	}
	return frames
}

// shortenSourcePath reduces "/abs/path/pkg/file.go:123" to "pkg/file.go:123",
// removing absolute local paths while keeping enough to locate the frame.
func shortenSourcePath(loc string) string {
	segs := strings.Split(loc, "/")
	if len(segs) > 2 {
		segs = segs[len(segs)-2:]
	}
	return strings.Join(segs, "/")
}

// changedAttributeNames returns the top-level schema attribute names that changed,
// as a non-nil slice. It appends only static schema keys, so no attribute value or
// map key can leak. Only Create and Update carry a diff; others return empty.
func changedAttributeNames(d *schema.ResourceData, resourceSchema map[string]*schema.Schema, op telemetry.Operation) []string {
	if d == nil || (op != telemetry.OperationCreate && op != telemetry.OperationUpdate) {
		return []string{}
	}
	return collectChangedNames(resourceSchema, d.HasChange)
}

// collectChangedNames is the core of changedAttributeNames, split out for
// deterministic testing. It can only ever return keys of resourceSchema.
func collectChangedNames(resourceSchema map[string]*schema.Schema, hasChange func(string) bool) []string {
	names := make([]string, 0)
	for name := range resourceSchema {
		if hasChange(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
