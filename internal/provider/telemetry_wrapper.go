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
	"runtime"
	"sort"
	"sync"
	"weak"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// Central client-analytics wrapping (TFCA-B3): replaces every managed resource's
// CRUD and import entry points, once, with wrappers that build a telemetry.Usage
// and hand it to a reporter — no resource_*.go file is touched. Reporting goes to
// a no-op sink until the TFCA-B5 transport and TFCA-B6 opt-out supply the real
// reporter, so today the wrapper does no I/O and changes no behavior. Panic
// recovery is deferred to TFCA-B4; a panic here propagates unchanged.

// telemetryReporter receives a fully-populated Usage. Report must not block the
// calling goroutine for long (the TFCA-B5 transport hands off to a worker pool).
type telemetryReporter interface {
	Report(telemetry.Usage)
}

// noopTelemetryReporter drops every Usage; it keeps this mechanism transparent
// until the TFCA-B5 transport replaces it.
type noopTelemetryReporter struct{}

func (noopTelemetryReporter) Report(telemetry.Usage) {}

// telemetryWrapConfig carries the process-scoped inputs used to populate a Usage.
type telemetryWrapConfig struct {
	// reporter receives each Usage; a nil reporter is treated as a no-op.
	reporter telemetryReporter

	// providerVersion is the released provider version.
	providerVersion string

	// terraformVersion is a func, not a value, because Core populates
	// schema.Provider.TerraformVersion during ConfigureProvider — after New
	// returns but before any CRUD/import call. A nil func yields "".
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
// on any later pass, so composing the wrap twice never double-emits. Keys are
// weak.Pointers so an entry does not keep the resource — or the provider graph
// its closures capture — alive; each resource registers a cleanup that drops its
// dead stub once collected, so repeated New() calls (e.g. confluent_tf_importer,
// one per apply) don't leak.
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

// wrapResourceForTelemetry wraps each entry point only when it is already
// non-nil. This matters: 8 managed resources have a nil UpdateContext, and the
// SDK treats a set update func as "updates allowed" — turning nil into a wrapper
// closure would nil-panic and falsely advertise those resources as updatable.
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

// wrapContextFunc wraps a Create/Read/Update/DeleteContext function. Sequence and
// timer are taken at entry (so order/duration are correct even if the report is
// dropped), and the Usage is built and reported synchronously after inner returns
// — the SDK reads ResourceData right after CRUD returns, so nothing may defer to
// a goroutine that touches d.
func wrapContextFunc(resourceType string, resourceSchema map[string]*schema.Schema, op telemetry.Operation, inner func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		diags := inner(ctx, d, meta)

		cfg.report(telemetry.Usage{
			RunID:             telemetry.RunID(),
			Sequence:          seq,
			StartedAt:         timer.StartTime(),
			DurationMs:        timer.Elapsed().Milliseconds(),
			OS:                runtime.GOOS,
			Arch:              runtime.GOARCH,
			ProviderVersion:   cfg.providerVersion,
			TerraformVersion:  cfg.tfVersion(),
			ResourceType:      resourceType,
			Operation:         op,
			ChangedAttributes: changedAttributeNames(d, resourceSchema, op),
			Error:             diags.HasError(),
		})
		return diags
	}
}

// wrapImportFunc wraps Importer.StateContext, which has a different signature and
// carries no attribute diff, so ChangedAttributes is always the empty slice.
func wrapImportFunc(resourceType string, resourceSchema map[string]*schema.Schema, inner schema.StateContextFunc, cfg telemetryWrapConfig) func(context.Context, *schema.ResourceData, interface{}) ([]*schema.ResourceData, error) {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
		seq := telemetry.NextSequence()
		timer := telemetry.StartTimer()

		imported, err := inner(ctx, d, meta)

		cfg.report(telemetry.Usage{
			RunID:             telemetry.RunID(),
			Sequence:          seq,
			StartedAt:         timer.StartTime(),
			DurationMs:        timer.Elapsed().Milliseconds(),
			OS:                runtime.GOOS,
			Arch:              runtime.GOARCH,
			ProviderVersion:   cfg.providerVersion,
			TerraformVersion:  cfg.tfVersion(),
			ResourceType:      resourceType,
			Operation:         telemetry.OperationImport,
			ChangedAttributes: []string{},
			Error:             err != nil,
		})
		return imported, err
	}
}

// changedAttributeNames returns the top-level schema attribute names that
// changed, as a non-nil slice (so it serializes as [] not null). It appends only
// static schema keys, so no attribute value or user-controlled map key can leak.
// Only Create and Update carry a diff; other operations report the empty slice.
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
