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
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// End-to-end validation of the enabled telemetry path: every event travels through the
// real bounded-worker transport (not a synchronous stand-in). Each test publishes an
// enabled runtime directly — exercising the real New()-wrapped resources without
// reverting the shipped test-mode gate — and runs serially, keeping one event in flight
// so the shallow, lossy transport queue never drops and delivery order matches invocation.

// e2eDeliveryTimeout bounds a single serialized delivery. Delivery is one local
// channel hop, so this is a deadlock backstop, not a timing assertion.
const e2eDeliveryTimeout = 5 * time.Second

// resourceOpKey identifies one emitted event by resource type and operation.
type resourceOpKey struct {
	resourceType string
	operation    telemetry.Operation
}

// enabledCapturingTransport publishes an enabled runtime whose sink is a real
// transport that forwards each delivered Usage to the returned channel. Capacity 1
// enforces the one-in-flight discipline the serialized sweeps rely on.
func enabledCapturingTransport(t *testing.T) <-chan telemetry.Usage {
	t.Helper()
	restorePublishedTelemetry(t)
	poster := capturingPoster{got: make(chan telemetry.Usage, 1)}
	transport := telemetry.NewTransport(poster, context.Background())
	t.Cleanup(transport.Close)
	publishedTelemetry.Store(&telemetryRuntime{config: telemetry.NewConfig(false), reporter: transport})
	return poster.got
}

// receiveUsage returns the single Usage delivered over the transport for one
// invocation, asserting its resource type, operation, and stable run ID. The timeout
// fails the test rather than hanging.
func receiveUsage(t *testing.T, got <-chan telemetry.Usage, resourceType string, op telemetry.Operation) telemetry.Usage {
	t.Helper()
	select {
	case u := <-got:
		if u.ResourceType != resourceType {
			t.Errorf("%s/%s: ResourceType = %q, want %q", resourceType, op, u.ResourceType, resourceType)
		}
		if u.Operation != op {
			t.Errorf("%s/%s: Operation = %q, want %q", resourceType, op, u.Operation, op)
		}
		if u.RunID != telemetry.RunID() {
			t.Errorf("%s/%s: RunID = %q, want the stable process run ID %q", resourceType, op, u.RunID, telemetry.RunID())
		}
		return u
	case <-time.After(e2eDeliveryTimeout):
		t.Fatalf("%s/%s: no telemetry reached the poster over the transport within %s", resourceType, op, e2eDeliveryTimeout)
		return telemetry.Usage{}
	}
}

// TestTelemetryE2E_AllManagedResourcesEmitCorrectPayloads drives every managed
// resource's wrapped entry points over the real transport (one event in flight) and
// asserts one correctly-typed event each, a stable run ID, and strictly increasing
// sequences. Nil args make the inner CRUD fail fast, so Error is not asserted.
func TestTelemetryE2E_AllManagedResourcesEmitCorrectPayloads(t *testing.T) {
	got := enabledCapturingTransport(t)
	p := New(testVersion, "")()
	ctx := context.Background()

	expected := map[resourceOpKey]bool{}
	seen := map[resourceOpKey]int{}
	var lastSeq int64
	total := 0
	var sample telemetry.Usage

	// drive invokes one entry point, then receives its single event before the next
	// invocation, keeping the shallow transport queue from ever overflowing.
	drive := func(resourceType string, op telemetry.Operation, call func()) {
		expected[resourceOpKey{resourceType, op}] = true
		call()
		u := receiveUsage(t, got, resourceType, op)
		total++
		seen[resourceOpKey{u.ResourceType, u.Operation}]++
		if u.Sequence <= lastSeq {
			t.Errorf("%s/%s: sequence %d is not strictly greater than the previous %d", resourceType, op, u.Sequence, lastSeq)
		}
		lastSeq = u.Sequence
		if sample.RunID == "" {
			sample = u
		}
	}

	for resourceType, r := range p.ResourcesMap {
		if r == nil {
			continue
		}
		if r.CreateContext != nil {
			drive(resourceType, telemetry.OperationCreate, func() { _ = r.CreateContext(ctx, nil, nil) })
		}
		if r.ReadContext != nil {
			drive(resourceType, telemetry.OperationRead, func() { _ = r.ReadContext(ctx, nil, nil) })
		}
		if r.UpdateContext != nil {
			drive(resourceType, telemetry.OperationUpdate, func() { _ = r.UpdateContext(ctx, nil, nil) })
		}
		if r.DeleteContext != nil {
			drive(resourceType, telemetry.OperationDelete, func() { _ = r.DeleteContext(ctx, nil, nil) })
		}
		if r.Importer != nil && r.Importer.StateContext != nil {
			drive(resourceType, telemetry.OperationImport, func() { _, _ = r.Importer.StateContext(ctx, nil, nil) })
		}
	}

	if len(p.ResourcesMap) < 60 {
		t.Fatalf("expected the full managed-resource catalog (~65), got %d", len(p.ResourcesMap))
	}
	if len(expected) == 0 {
		t.Fatal("no managed resources were driven; ResourcesMap looks empty")
	}
	if total != len(expected) {
		t.Errorf("received %d events, want %d (one per wrapped entry point)", total, len(expected))
	}
	for key := range expected {
		if seen[key] != 1 {
			t.Errorf("%s/%s: received %d events, want exactly 1", key.resourceType, key.operation, seen[key])
		}
	}
	for key := range seen {
		if !expected[key] {
			t.Errorf("%s/%s: unexpected event for an entry point that was not driven", key.resourceType, key.operation)
		}
	}

	t.Logf("validated %d managed resources / %d wrapped entry points over the real transport; run_id=%s (stable, strictly increasing sequences)",
		len(p.ResourcesMap), total, telemetry.RunID())
	t.Logf("sample delivered payload: resource_type=%s operation=%s run_id=%s sequence=%d os=%s arch=%s provider_version=%s error=%v",
		sample.ResourceType, sample.Operation, sample.RunID, sample.Sequence, sample.OS, sample.Arch, sample.ProviderVersion, sample.Error)
}

// TestTelemetryE2E_NamedResourcesLifecycleOverTransport drives a control-plane
// (confluent_environment) and a data-plane (confluent_kafka_topic) resource through all
// five operations over the same real transport, asserting each delivers one event with
// the correct resource type, operation, stable run ID, and a strictly increasing sequence.
func TestTelemetryE2E_NamedResourcesLifecycleOverTransport(t *testing.T) {
	got := enabledCapturingTransport(t)
	p := New(testVersion, "")()
	ctx := context.Background()

	for _, resourceType := range []string{"confluent_environment", "confluent_kafka_topic"} {
		r, ok := p.ResourcesMap[resourceType]
		if !ok {
			t.Fatalf("%s is not a managed resource", resourceType)
		}
		if r.CreateContext == nil || r.ReadContext == nil || r.UpdateContext == nil ||
			r.DeleteContext == nil || r.Importer == nil || r.Importer.StateContext == nil {
			t.Fatalf("%s must declare all five entry points for a full-lifecycle validation", resourceType)
		}

		lifecycle := []struct {
			op   telemetry.Operation
			call func()
		}{
			{telemetry.OperationCreate, func() { _ = r.CreateContext(ctx, nil, nil) }},
			{telemetry.OperationRead, func() { _ = r.ReadContext(ctx, nil, nil) }},
			{telemetry.OperationUpdate, func() { _ = r.UpdateContext(ctx, nil, nil) }},
			{telemetry.OperationDelete, func() { _ = r.DeleteContext(ctx, nil, nil) }},
			{telemetry.OperationImport, func() { _, _ = r.Importer.StateContext(ctx, nil, nil) }},
		}

		var lastSeq int64
		for _, step := range lifecycle {
			step.call()
			u := receiveUsage(t, got, resourceType, step.op)
			if u.Sequence <= lastSeq {
				t.Errorf("%s/%s: sequence %d is not strictly greater than the previous %d", resourceType, step.op, u.Sequence, lastSeq)
			}
			lastSeq = u.Sequence
			t.Logf("delivered over transport: resource_type=%s operation=%s run_id=%s sequence=%d error=%v",
				u.ResourceType, u.Operation, u.RunID, u.Sequence, u.Error)
		}
	}
}

// TestTelemetryE2E_ForcedPanicProducesCrashPayload forces a panic in a wrapped
// resource and asserts the crash payload travels end to end through the enabled
// reporter and the real transport: the delivered event carries error=true, a trimmed
// stack trace, and the correct resource type and operation.
func TestTelemetryE2E_ForcedPanicProducesCrashPayload(t *testing.T) {
	got := enabledCapturingTransport(t)

	const resourceType = "confluent_panic_probe"
	panicking := &schema.Resource{
		Schema: map[string]*schema.Schema{"name": {Type: schema.TypeString, Optional: true}},
		CreateContext: func(context.Context, *schema.ResourceData, interface{}) diag.Diagnostics {
			panic("forced panic for crash-payload validation")
		},
	}
	// Wrap exactly as New() does, with the published reporter as the sink.
	wrapResourcesMapForTelemetry(map[string]*schema.Resource{resourceType: panicking}, telemetryWrapConfig{
		reporter:         publishedTelemetryReporter{},
		providerVersion:  "9.9.9-test",
		terraformVersion: func() string { return "1.7.0-test" },
	})

	// The wrapper recovers the panic and returns error diagnostics.
	if diags := panicking.CreateContext(context.Background(), nil, nil); !diags.HasError() {
		t.Fatal("expected error diagnostics from the recovered panic")
	}

	select {
	case u := <-got:
		if u.ResourceType != resourceType {
			t.Errorf("ResourceType = %q, want %q", u.ResourceType, resourceType)
		}
		if u.Operation != telemetry.OperationCreate {
			t.Errorf("Operation = %q, want CREATE", u.Operation)
		}
		if !u.Error {
			t.Error("Error = false, want true for a crash payload")
		}
		if len(u.StackFrames) == 0 {
			t.Error("StackFrames is empty, want a trimmed stack trace on the crash payload")
		}
	case <-time.After(e2eDeliveryTimeout):
		t.Fatal("no crash payload reached the poster over the transport")
	}
}
