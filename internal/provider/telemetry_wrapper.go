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
// The wrappers are also the provider's only panic recovery: a panic in a wrapped
// CRUD or import call is recovered and returned as an error (with the trimmed
// stack in its detail) instead of crashing the process, and is reported as a
// crash event. Building and reporting the Usage has its own recover, so a panic
// while doing that is contained too.

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

		// Recover a panic from the CRUD call and return it as error diagnostics
		// instead of letting it crash the process.
		rec, stack := runGuarded(func() { diags = inner(ctx, d, meta) })
		if rec != nil {
			diags = panicDiagnostics(resourceType, op, rec, stack)
		}

		// Build and report the Usage under a separate recover, so a panic while
		// building or sending it is also contained.
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

		// Recover an import panic as an error, discarding any partial result.
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

// buildUsage assembles a Usage. stack is set only on the panic path; a normal
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

// runGuarded runs fn and recovers any panic, returning the recovered value (nil
// if fn returned normally) and a trimmed stack trace captured at the panic site.
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

// reportSafely builds and reports a Usage, swallowing any panic so telemetry can
// never crash the process it observes.
func reportSafely(cfg telemetryWrapConfig, build func() telemetry.Usage) {
	defer func() { _ = recover() }()
	cfg.report(build())
}

// panicDiagnostics converts a recovered panic into error diagnostics, with the
// stack in the detail (see panicDetail).
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

// panicDetail renders the panic value and its trimmed stack for the operator's
// error, so a recovered panic stays as diagnosable as the crash it replaces. It
// shows at most maxDetailFrames frames: the top frames already include the panic
// origin, and a longer trace only buries it in the terraform output. The fuller
// trace (up to maxStackFrames) still goes to the crash telemetry for triage.
func panicDetail(rec interface{}, stack []string) string {
	if len(stack) == 0 {
		return fmt.Sprintf("%v", rec)
	}
	shown, suffix := stack, ""
	if len(shown) > maxDetailFrames {
		suffix = fmt.Sprintf("\n... (showing top %d of %d stack frames)", maxDetailFrames, len(shown))
		shown = shown[:maxDetailFrames]
	}
	return fmt.Sprintf("%v\n\n%s%s", rec, strings.Join(shown, "\n"), suffix)
}

// maxStackFrames caps how many frames a crash report carries to the backend.
const maxStackFrames = 50

// maxDetailFrames caps how many frames the operator-facing error Detail shows.
// It is deliberately smaller than maxStackFrames: the crash telemetry keeps the
// fuller trace for backend triage/dedup, while the operator sees a readable
// slice whose top frames already include the panic origin.
const maxDetailFrames = 15

// trimmedStackFrames captures the current stack and reduces it to file:line
// frames, stripping absolute paths so no local path (e.g. /Users/<name>/...) is
// exposed.
func trimmedStackFrames() []string {
	lines := strings.Split(string(debug.Stack()), "\n")
	frames := make([]string, 0, len(lines))
	for _, line := range lines {
		loc := strings.TrimSpace(line)
		if !strings.Contains(loc, ".go:") {
			continue
		}
		// Drop the trailing " +0x.." offset only (not everything after the first
		// space), so a path containing a space is not truncated.
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
// dropping the absolute path while keeping enough to locate the frame. Backslashes
// are normalized to "/" first so a Windows-style path is redacted the same way.
func shortenSourcePath(loc string) string {
	loc = strings.ReplaceAll(loc, "\\", "/")
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
