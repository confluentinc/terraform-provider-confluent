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
	"fmt"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// This file is the central client-analytics wrapping mechanism (TFCA-B3). It
// replaces every managed resource's CRUD and import entry points, once, with
// wrappers that build a telemetry.Usage and hand it to a reporter. It is the
// single place this happens: no individual resource_*.go file is touched.
//
// Scope boundaries with the rest of Epic B, so this file stays reviewable in
// isolation:
//   - The payload/timing/run-ID model it populates lives in the telemetry
//     package (TFCA-B1).
//   - Reporting here goes to a no-op sink until the wiring lands. The
//     bounded-worker network transport (TFCA-B5) supplies the real reporter, and
//     the opt-out wiring (TFCA-B6) decides when reporting is enabled.
//   - Panic recovery (TFCA-B4) lives here: a panic in a wrapped CRUD or import
//     call is recovered, converted to an error result, and reported as a crash
//     event rather than crashing the provider process. The report path is
//     guarded separately so a panic while building or enqueuing the payload is
//     also contained.

// telemetryReporter receives a fully-populated Usage for one resource
// operation. Report must not block the calling CRUD/import goroutine for long:
// the no-op shipped here returns immediately, and the TFCA-B5 transport that
// replaces it hands off to a bounded worker pool.
type telemetryReporter interface {
	Report(telemetry.Usage)
}

// noopTelemetryReporter drops every Usage. It is the reporter used until the
// TFCA-B5 transport replaces it, which is what makes the wrapping mechanism
// introduced here behaviorally transparent.
type noopTelemetryReporter struct{}

func (noopTelemetryReporter) Report(telemetry.Usage) {}

// telemetryWrapConfig carries the process-scoped inputs the wrapper needs to
// populate a Usage.
type telemetryWrapConfig struct {
	// reporter receives each Usage. A nil reporter is treated as a no-op.
	reporter telemetryReporter

	// providerVersion is the released provider version, constant for the life of
	// the process (the value threaded through New).
	providerVersion string

	// terraformVersion returns the version of Terraform Core that launched the
	// provider. It is a function, not a value, because Terraform Core populates
	// schema.Provider.TerraformVersion during ConfigureProvider — after New
	// returns but before any CRUD/import call runs. A nil function yields "".
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

// wrappedResources records which *schema.Resource values have already had their
// entry points wrapped. Composing the wrap pass more than once over the same map
// would double-emit events and, once TFCA-B4 lands, stack recover layers; this
// guard makes a second pass a no-op. It is enforced in code rather than by a
// "wrap only once" comment so it survives a future refactor that reorders or
// repeats the call. Resource pointers are unique per constructed provider (each
// xResource() constructor returns a fresh *schema.Resource), so resources from
// different providers never collide here and one provider's wrapping never
// suppresses another's.
var wrappedResources sync.Map // map[*schema.Resource]struct{}

// wrapResourcesMapForTelemetry wraps the CRUD and import entry points of every
// resource in the map. It must run after ResourcesMap is fully constructed and
// before the provider is served. It is safe to call more than once; each
// resource is wrapped at most once.
func wrapResourcesMapForTelemetry(resources map[string]*schema.Resource, cfg telemetryWrapConfig) {
	if cfg.reporter == nil {
		cfg.reporter = noopTelemetryReporter{}
	}
	for resourceType, r := range resources {
		if r == nil {
			continue
		}
		if _, alreadyWrapped := wrappedResources.LoadOrStore(r, struct{}{}); alreadyWrapped {
			continue
		}
		wrapResourceForTelemetry(resourceType, r, cfg)
	}
}

// resourceWrapped reports whether r has been through wrapResourcesMapForTelemetry.
// It exists for tests that assert coverage of the wrap pass.
func resourceWrapped(r *schema.Resource) bool {
	_, ok := wrappedResources.Load(r)
	return ok
}

// wrapResourceForTelemetry wraps each entry point of a single resource
// independently, and only when that entry point is already non-nil.
//
// The non-nil guard matters: 8 of the managed resources have no UpdateContext.
// The SDK decides at runtime whether an update is legal by checking whether an
// update function is set (helper/schema/resource.go), so replacing a nil
// UpdateContext with a non-nil wrapper closure would both nil-panic when that
// closure invoked the (nil) inner function and silently make those resources
// report as updatable. The same reasoning applies to any resource that happens
// to leave a CRUD entry point unset.
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

// wrapContextFunc wraps a Create/Read/Update/DeleteContext function. The return
// type is the bare function type rather than a named schema.*ContextFunc so the
// one wrapper is assignable to all four fields (they share this underlying
// signature).
//
// The sequence number and timer are taken at entry, before inner runs, so the
// recorded order reflects invocation order even for operations whose report is
// later dropped, and the duration brackets exactly the inner call. The Usage is
// built and reported synchronously after inner returns and before the wrapper
// returns, because the SDK reads the *schema.ResourceData immediately after the
// CRUD function returns — nothing here may defer to a background goroutine that
// touches d.
func wrapContextFunc(resourceType string, resourceSchema map[string]*schema.Schema, op telemetry.Operation, inner func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) (diags diag.Diagnostics) {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		// Recover a panic from the wrapped CRUD call (TFCA-B4). This is the only
		// panic recovery in the provider: without it a panic in any resource
		// crashes the whole provider process. On panic we convert to error
		// diagnostics and still report a crash event, rather than propagating.
		rec, stack := runGuarded(func() { diags = inner(ctx, d, meta) })
		if rec != nil {
			diags = panicDiagnostics(resourceType, op, rec)
		}

		// Build and report the Usage under its own recover, separate from the one
		// above: the one above only guards the CRUD call, so a panic while
		// reflecting over ResourceData or formatting the payload here would
		// otherwise escape uncaught and crash the very process this feature
		// protects. The transport enqueue is non-blocking, so this returns
		// promptly even if the backend is hung.
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

// wrapImportFunc wraps an Importer.StateContext function. Import has a different
// signature (it returns the imported resource data and an error, not
// diagnostics), so it is wrapped separately from the CRUD entry points. Import
// carries no attribute-level diff, so ChangedAttributes is always the empty
// (non-nil) slice. Panics are recovered on the same terms as the CRUD path.
func wrapImportFunc(resourceType string, resourceSchema map[string]*schema.Schema, inner schema.StateContextFunc, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) ([]*schema.ResourceData, error) {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) (imported []*schema.ResourceData, err error) {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		rec, stack := runGuarded(func() { imported, err = inner(ctx, d, meta) })
		if rec != nil {
			imported = nil
			err = panicError(resourceType, telemetry.OperationImport, rec)
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
// if fn returned normally) and, on panic, a trimmed stack trace captured at the
// point of recovery.
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
// place of the wrapped CRUD call's result.
func panicDiagnostics(resourceType string, op telemetry.Operation, rec interface{}) diag.Diagnostics {
	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("the Confluent provider recovered from a panic during %s of %s", strings.ToLower(string(op)), resourceType),
		Detail:   fmt.Sprintf("%v", rec),
	}}
}

// panicError is the import-path equivalent of panicDiagnostics.
func panicError(resourceType string, op telemetry.Operation, rec interface{}) error {
	return fmt.Errorf("the Confluent provider recovered from a panic during %s of %s: %v", strings.ToLower(string(op)), resourceType, rec)
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
		// Drop the trailing " +0x.." program-counter offset.
		if sp := strings.IndexByte(loc, ' '); sp >= 0 {
			loc = loc[:sp]
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

// changedAttributeNames returns the schema-declared, top-level attribute names
// that changed in this operation. It returns a non-nil slice (empty when nothing
// changed) so the field serializes as [] rather than null.
//
// It iterates the resource's own schema keys and reports only those names — it
// can never emit a map attribute's dynamic keys or any attribute value, because
// the only strings it ever appends are the static schema keys themselves. A
// schema.TypeMap attribute therefore contributes only its own name (e.g.
// "config"), never the keys of the map a user set.
//
// Only Create and Update carry a meaningful attribute diff; Read, Delete, and
// Import report the empty slice.
func changedAttributeNames(d *schema.ResourceData, resourceSchema map[string]*schema.Schema, op telemetry.Operation) []string {
	if d == nil || (op != telemetry.OperationCreate && op != telemetry.OperationUpdate) {
		return []string{}
	}
	return collectChangedNames(resourceSchema, d.HasChange)
}

// collectChangedNames is the leak-proof core of changedAttributeNames, split out
// so it can be tested deterministically without constructing a diff-bearing
// *schema.ResourceData. The only strings it can ever return are keys of
// resourceSchema — the statically declared attribute names — so no attribute
// value and no user-controlled map key can escape into a Usage, regardless of
// what hasChange reports.
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
