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
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// recordingReporter is a concurrency-safe telemetryReporter that keeps every
// Usage it is handed, for assertions.
type recordingReporter struct {
	mu    sync.Mutex
	usage []telemetry.Usage
}

func (r *recordingReporter) Report(u telemetry.Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.usage = append(r.usage, u)
}

func (r *recordingReporter) snapshot() []telemetry.Usage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]telemetry.Usage, len(r.usage))
	copy(out, r.usage)
	return out
}

func (r *recordingReporter) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.usage)
}

// newTestResource builds a minimal managed resource whose CRUD and import entry
// points are transparent no-ops, so the only observable effect of invoking them
// is whatever the telemetry wrapper adds.
func newTestResource() *schema.Resource {
	okDiag := func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics { return nil }
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"name": {Type: schema.TypeString, Optional: true},
		},
		CreateContext: okDiag,
		ReadContext:   okDiag,
		UpdateContext: okDiag,
		DeleteContext: okDiag,
		Importer: &schema.ResourceImporter{
			StateContext: func(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				return []*schema.ResourceData{d}, nil
			},
		},
	}
}

func testWrapConfig(reporter telemetryReporter) telemetryWrapConfig {
	return telemetryWrapConfig{
		reporter:         reporter,
		providerVersion:  "9.9.9-test",
		terraformVersion: func() string { return "1.7.0-test" },
	}
}

// TestWrapResourcesMap_CoversAllManagedResourcesAndExcludesDataSources asserts
// the central pass wraps every managed resource (v1 scope) and leaves every data
// source untouched, with no per-resource file edits.
func TestWrapResourcesMap_CoversAllManagedResourcesAndExcludesDataSources(t *testing.T) {
	p := New(testVersion, "")()

	// v1 scope is the 64 managed resources; the 63 data sources are excluded.
	// This count is pinned deliberately: adding a managed resource should be a
	// conscious decision to extend telemetry coverage, so bump it here when the
	// ResourcesMap grows.
	const wantManagedResources = 64
	if got := len(p.ResourcesMap); got != wantManagedResources {
		t.Errorf("ResourcesMap has %d resources, want %d (update this pin if resources were intentionally added/removed)", got, wantManagedResources)
	}

	for name, r := range p.ResourcesMap {
		if !resourceWrapped(r) {
			t.Errorf("managed resource %q was not wrapped for telemetry", name)
		}
	}
	for name, ds := range p.DataSourcesMap {
		if resourceWrapped(ds) {
			t.Errorf("data source %q was wrapped for telemetry but data sources are excluded in v1", name)
		}
	}
}

// TestWrapResourcesMap_PreservesNilUpdateContext asserts a resource with no
// UpdateContext still has a nil UpdateContext after wrapping — never a non-nil
// no-op closure that would nil-panic and change the SDK's update-legality check.
func TestWrapResourcesMap_PreservesNilUpdateContext(t *testing.T) {
	// The real provider's known nil-update resources, at the time of writing.
	// Used only as a readable cross-check; the load-bearing assertion below is
	// the count, which needs no name maintenance.
	knownNilUpdate := map[string]bool{
		"confluent_byok_key":                        true,
		"confluent_connect_artifact":                true,
		"confluent_custom_connector_plugin_version": true,
		"confluent_flink_artifact":                  true,
		"confluent_ksql_cluster":                    true,
		"confluent_provider_integration":            true,
		"confluent_provider_integration_setup":      true,
		"confluent_tf_importer":                     true,
	}

	p := New(testVersion, "")()
	var gotNilUpdate []string
	for name, r := range p.ResourcesMap {
		if r.UpdateContext == nil {
			gotNilUpdate = append(gotNilUpdate, name)
			// A nil-update resource must still have its other entry points intact.
			if r.CreateContext == nil || r.ReadContext == nil || r.DeleteContext == nil {
				t.Errorf("resource %q: a non-update entry point became nil after wrapping", name)
			}
		}
	}

	// If wrapping had turned any nil UpdateContext into a non-nil no-op closure,
	// this count would drop below the known set.
	if len(gotNilUpdate) != len(knownNilUpdate) {
		t.Errorf("found %d resources with nil UpdateContext (%v), want %d", len(gotNilUpdate), gotNilUpdate, len(knownNilUpdate))
	}
	for _, name := range gotNilUpdate {
		if !knownNilUpdate[name] {
			t.Errorf("resource %q has nil UpdateContext but is not in the known set — update the cross-check", name)
		}
	}

	// Also assert at the mechanism level, independent of the resource catalog: a
	// synthetic resource with a nil UpdateContext stays nil.
	r := newTestResource()
	r.UpdateContext = nil
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"x": r}, testWrapConfig(&recordingReporter{}))
	if r.UpdateContext != nil {
		t.Errorf("synthetic resource: nil UpdateContext became non-nil after wrapping")
	}
	if r.CreateContext == nil {
		t.Errorf("synthetic resource: CreateContext became nil after wrapping")
	}
}

// TestWrapResourcesMap_OnlyContextEntryPointsAreUsed guards the assumption the
// wrapper relies on: every managed resource uses only the *Context entry points.
// If a future resource uses a deprecated (Create) or WithoutTimeout variant, it
// would silently escape telemetry; this fails the build so the wrapper is
// extended deliberately.
func TestWrapResourcesMap_OnlyContextEntryPointsAreUsed(t *testing.T) {
	p := New(testVersion, "")()
	for name, r := range p.ResourcesMap {
		if r.Create != nil || r.Read != nil || r.Update != nil || r.Delete != nil {
			t.Errorf("resource %q uses a deprecated non-context CRUD func; extend the telemetry wrapper", name)
		}
		if r.CreateWithoutTimeout != nil || r.ReadWithoutTimeout != nil || r.UpdateWithoutTimeout != nil || r.DeleteWithoutTimeout != nil {
			t.Errorf("resource %q uses a *WithoutTimeout CRUD func; extend the telemetry wrapper", name)
		}
		// The import wrapper only wraps Importer.StateContext; a resource using
		// the deprecated Importer.State would silently escape import telemetry.
		if r.Importer != nil && r.Importer.State != nil {
			t.Errorf("resource %q uses the deprecated Importer.State; extend the telemetry wrapper to wrap it", name)
		}
	}
}

// TestWrapper_TransparentOnHappyPath asserts wrapped CRUD is behaviorally
// transparent: the inner return value passes through unchanged, and exactly one
// event is emitted per call carrying the right operation, run ID, and metadata.
func TestWrapper_TransparentOnHappyPath(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()

	sentinel := diag.Diagnostics{{Severity: diag.Warning, Summary: "hello"}}
	r.CreateContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics { return sentinel }

	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

	got := r.CreateContext(context.Background(), nil, nil)
	if len(got) != 1 || got[0].Summary != "hello" {
		t.Fatalf("wrapper did not pass inner diagnostics through unchanged: %+v", got)
	}
	if rec.count() != 1 {
		t.Fatalf("expected exactly 1 telemetry event, got %d", rec.count())
	}
	u := rec.snapshot()[0]
	if u.ResourceType != "confluent_thing" {
		t.Errorf("ResourceType = %q, want confluent_thing", u.ResourceType)
	}
	if u.Operation != telemetry.OperationCreate {
		t.Errorf("Operation = %q, want CREATE", u.Operation)
	}
	if u.RunID != telemetry.RunID() {
		t.Errorf("RunID = %q, want process run ID %q", u.RunID, telemetry.RunID())
	}
	if u.ProviderVersion != "9.9.9-test" || u.TerraformVersion != "1.7.0-test" {
		t.Errorf("version metadata not populated: provider=%q terraform=%q", u.ProviderVersion, u.TerraformVersion)
	}
	if u.Error {
		t.Errorf("Error should be false for a warning-only result")
	}
	if u.ChangedAttributes == nil {
		t.Errorf("ChangedAttributes must be non-nil so it serializes as [] not null")
	}
}

// TestWrapper_ErrorFlag asserts the Error flag reflects an error-severity result.
func TestWrapper_ErrorFlag(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	r.DeleteContext = func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
		return diag.Errorf("boom")
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

	_ = r.DeleteContext(context.Background(), nil, nil)
	u := rec.snapshot()[0]
	if !u.Error {
		t.Errorf("Error flag should be true when the operation returns error diagnostics")
	}
	if u.Operation != telemetry.OperationDelete {
		t.Errorf("Operation = %q, want DELETE", u.Operation)
	}
}

// TestWrapper_ImportWrappedSeparately asserts the import path is wrapped with the
// correct signature and reports an IMPORT event while passing results through.
func TestWrapper_ImportWrappedSeparately(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_thing": r}, testWrapConfig(rec))

	out, err := r.Importer.StateContext(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("import returned unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("import passthrough returned %d results, want 1", len(out))
	}
	if rec.count() != 1 {
		t.Fatalf("import emitted %d events, want exactly 1", rec.count())
	}
	u := rec.snapshot()[0]
	if u.Operation != telemetry.OperationImport {
		t.Errorf("Operation = %q, want IMPORT", u.Operation)
	}
	if u.ChangedAttributes == nil || len(u.ChangedAttributes) != 0 {
		t.Errorf("import ChangedAttributes should be empty non-nil, got %#v", u.ChangedAttributes)
	}
}

// TestWrapper_DoubleWrapEmitsOneEventPerCall asserts running the wrap pass twice
// over the same map does not compose wrappers: each CRUD call still emits exactly
// one event.
func TestWrapper_DoubleWrapEmitsOneEventPerCall(t *testing.T) {
	rec := &recordingReporter{}
	r := newTestResource()
	m := map[string]*schema.Resource{"confluent_thing": r}

	wrapResourcesMapForTelemetry(m, testWrapConfig(rec))
	wrapResourcesMapForTelemetry(m, testWrapConfig(rec)) // second pass must be a no-op

	_ = r.CreateContext(context.Background(), nil, nil)
	if rec.count() != 1 {
		t.Fatalf("double-wrapped Create emitted %d events, want exactly 1", rec.count())
	}
}

// TestWrapper_ConcurrentRunIDsAndSequences drives ~20 independent resources'
// entry points concurrently — modeling Terraform's parallel graph walk — and
// asserts the run ID is stable and every sequence number is unique. Run with
// -race to catch data races in the process-scoped run ID and sequence counter.
func TestWrapper_ConcurrentRunIDsAndSequences(t *testing.T) {
	rec := &recordingReporter{}
	const numResources = 20

	m := make(map[string]*schema.Resource, numResources)
	for i := 0; i < numResources; i++ {
		m[fmt.Sprintf("confluent_thing_%d", i)] = newTestResource()
	}
	wrapResourcesMapForTelemetry(m, testWrapConfig(rec))

	// Collect every entry point to invoke; ~20 resources x 4 CRUD + 1 import.
	type call func()
	var calls []call
	for _, r := range m {
		r := r
		calls = append(calls,
			func() { r.CreateContext(context.Background(), nil, nil) },
			func() { r.ReadContext(context.Background(), nil, nil) },
			func() { r.UpdateContext(context.Background(), nil, nil) },
			func() { r.DeleteContext(context.Background(), nil, nil) },
			func() { r.Importer.StateContext(context.Background(), nil, nil) },
		)
	}

	var wg sync.WaitGroup
	for _, c := range calls {
		wg.Add(1)
		go func(c call) {
			defer wg.Done()
			c()
		}(c)
	}
	wg.Wait()

	events := rec.snapshot()
	if len(events) != len(calls) {
		t.Fatalf("emitted %d events, want %d (one per invocation)", len(events), len(calls))
	}

	runID := telemetry.RunID()
	seen := make(map[int64]bool, len(events))
	for _, u := range events {
		if u.RunID != runID {
			t.Errorf("event has RunID %q, want stable process run ID %q", u.RunID, runID)
		}
		if u.Sequence <= 0 {
			t.Errorf("sequence %d is not positive", u.Sequence)
		}
		if seen[u.Sequence] {
			t.Errorf("duplicate sequence number %d — counter is not race-free", u.Sequence)
		}
		seen[u.Sequence] = true
	}
	if len(seen) != len(events) {
		t.Errorf("got %d distinct sequence numbers across %d events", len(seen), len(events))
	}
}

// TestCollectChangedNames_NamesOnlyNeverMapKeys asserts the changed-attribute
// computation can only ever emit statically declared attribute names — never a
// user-controlled map key or value — even when the change detector claims a
// nested map key changed. This is the provider-side guarantee that pairs with
// the TFCA-B2 build-time allowlist.
func TestCollectChangedNames_NamesOnlyNeverMapKeys(t *testing.T) {
	// Schema modeled on confluent_kafka_topic: a TypeMap "config" plus scalars.
	resourceSchema := map[string]*schema.Schema{
		"config":        {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
		"topic_name":    {Type: schema.TypeString, Required: true},
		"partitions":    {Type: schema.TypeInt, Optional: true},
		"never_touched": {Type: schema.TypeString, Optional: true},
	}

	// A detector that reports EVERYTHING as changed, including a runtime map key
	// that must never leak. The map key is never queried (the function only
	// iterates schema keys), so it cannot appear in the output.
	hasChange := func(name string) bool { return true }

	got := collectChangedNames(resourceSchema, hasChange)

	want := map[string]bool{"config": true, "topic_name": true, "partitions": true, "never_touched": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want the 4 declared names only", got)
	}
	for _, name := range got {
		if !want[name] {
			t.Errorf("emitted %q which is not a declared schema attribute name", name)
		}
		// Defense-in-depth: a leaked map key would contain a '.' address like
		// "config.cleanup.policy". None of the declared names do.
		if name == "config.cleanup.policy" || name == "cleanup.policy" {
			t.Errorf("emitted a runtime map key %q — map contents must never leak", name)
		}
	}

	// And when nothing changed, the result is an empty, non-nil slice.
	none := collectChangedNames(resourceSchema, func(string) bool { return false })
	if none == nil {
		t.Errorf("collectChangedNames must return non-nil even when nothing changed")
	}
	if len(none) != 0 {
		t.Errorf("expected no changed names, got %v", none)
	}
}

// TestWrapper_ChangedAttributesFromRealResourceData exercises the full
// changed-attributes path end to end with a genuine diff-bearing
// *schema.ResourceData rather than a stubbed detector, proving that a
// user-controlled map attribute contributes only its declared name to a Usage —
// never its runtime keys. This is the empirical counterpart to the structural
// TestCollectChangedNames_NamesOnlyNeverMapKeys.
func TestWrapper_ChangedAttributesFromRealResourceData(t *testing.T) {
	rec := &recordingReporter{}
	// Schema modeled on confluent_kafka_topic: a TypeMap "config" plus a scalar.
	r := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"config":     {Type: schema.TypeMap, Optional: true, Elem: &schema.Schema{Type: schema.TypeString}},
			"topic_name": {Type: schema.TypeString, Optional: true},
		},
		CreateContext: func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics { return nil },
	}
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{"confluent_kafka_topic": r}, testWrapConfig(rec))

	// A create-style ResourceData whose config map carries a user key
	// ("cleanup.policy") that must never leak into telemetry.
	d := schema.TestResourceDataRaw(t, r.Schema, map[string]interface{}{
		"config":     map[string]interface{}{"cleanup.policy": "compact"},
		"topic_name": "orders",
	})

	_ = r.CreateContext(context.Background(), d, nil)

	u := rec.snapshot()[0]
	want := []string{"config", "topic_name"}
	if !reflect.DeepEqual(u.ChangedAttributes, want) {
		t.Fatalf("ChangedAttributes = %#v, want %v (declared top-level names only)", u.ChangedAttributes, want)
	}
	for _, name := range u.ChangedAttributes {
		if strings.Contains(name, ".") {
			t.Errorf("ChangedAttributes leaked a nested/map key: %q", name)
		}
	}
}

// TestChangedAttributeNames_NonCreateUpdateAreEmpty asserts read/delete/import
// report an empty (non-nil) changed-attributes slice and never dereference a nil
// ResourceData.
func TestChangedAttributeNames_NonCreateUpdateAreEmpty(t *testing.T) {
	resourceSchema := map[string]*schema.Schema{"name": {Type: schema.TypeString}}
	for _, op := range []telemetry.Operation{telemetry.OperationRead, telemetry.OperationDelete, telemetry.OperationImport} {
		got := changedAttributeNames(nil, resourceSchema, op)
		if got == nil {
			t.Errorf("op %s: changedAttributeNames returned nil, want empty non-nil slice", op)
		}
		if len(got) != 0 {
			t.Errorf("op %s: expected empty changed attributes, got %v", op, got)
		}
	}
	// Even Create/Update with a nil ResourceData must not panic.
	if got := changedAttributeNames(nil, resourceSchema, telemetry.OperationCreate); len(got) != 0 {
		t.Errorf("nil ResourceData on create should yield empty changed attributes, got %v", got)
	}
}
