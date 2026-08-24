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
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Acceptance criterion: a single run ID is shared across every resource
// operation in one provider process. RunID() must return the same, valid,
// non-empty value on every call.
func TestRunID_StableAndValid(t *testing.T) {
	first := RunID()
	if first == "" {
		t.Fatal("RunID() returned an empty string")
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Fatalf("RunID() = %q is not a valid UUID: %v", first, err)
	}
	for i := 0; i < 1000; i++ {
		if got := RunID(); got != first {
			t.Fatalf("RunID() not stable across calls: first=%q got=%q", first, got)
		}
	}
}

// Under Terraform's parallel graph walk many goroutines read the run ID
// concurrently. Every one must observe the same value, and -race must stay
// clean.
func TestRunID_ConcurrentStable(t *testing.T) {
	const goroutines = 64
	results := make([]string, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = RunID()
		}(i)
	}
	wg.Wait()

	want := RunID()
	for i, got := range results {
		if got != want {
			t.Fatalf("goroutine %d observed RunID %q, want %q", i, got, want)
		}
	}
}

// Acceptance criterion: sequence numbers are monotonic. Consecutive calls from
// a single goroutine must increase by exactly one. This holds regardless of the
// counter's absolute starting point (other tests in the package also advance
// it), because the assertion is relative to the first value observed here.
//
// This and the concurrent test below share the process-global counter and must
// not call t.Parallel(): they rely on no other test advancing it during their
// run.
func TestNextSequence_MonotonicSequential(t *testing.T) {
	first := NextSequence()
	for i := int64(1); i <= 1000; i++ {
		got := NextSequence()
		if got != first+i {
			t.Fatalf("NextSequence() = %d, want %d", got, first+i)
		}
	}
}

// Acceptance criterion: sequence numbers are race-free under a parallel graph
// walk (run with -race). Every value allocated across all goroutines must be
// unique, and the full set must be gap-free — proving each increment was
// atomic. The uniqueness and count assertions are order-independent; the
// gap-free (contiguity) assertion additionally assumes no other goroutine
// advances the counter during this test, which holds only while the sequence
// tests stay serial (see the note on the sequential test above — no
// t.Parallel()).
func TestNextSequence_ConcurrentUniqueAndContiguous(t *testing.T) {
	const goroutines = 16
	const perGoroutine = 500
	const total = goroutines * perGoroutine

	out := make(chan int64, total)
	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				out <- NextSequence()
			}
		}()
	}
	wg.Wait()
	close(out)

	seen := make(map[int64]struct{}, total)
	var min, max int64
	first := true
	for v := range out {
		if _, dup := seen[v]; dup {
			t.Fatalf("duplicate sequence number %d — increment was not atomic", v)
		}
		seen[v] = struct{}{}
		if first {
			min, max, first = v, v, false
			continue
		}
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if len(seen) != total {
		t.Fatalf("got %d unique sequence numbers, want %d", len(seen), total)
	}
	if max-min+1 != int64(total) {
		t.Fatalf("sequence numbers not contiguous: min=%d max=%d span=%d, want span %d",
			min, max, max-min+1, total)
	}
}

func TestTimer(t *testing.T) {
	before := time.Now()
	tm := StartTimer()
	after := time.Now()

	if tm.StartTime().Before(before) || tm.StartTime().After(after) {
		t.Fatalf("StartTime %v not within [%v, %v]", tm.StartTime(), before, after)
	}

	// Lower bound only: time.Sleep guarantees at least `sleep` elapses, so this
	// rejects a constant or clock-ignoring Elapsed() while staying non-flaky
	// under CI load (no upper bound, which would be load-sensitive).
	const sleep = 5 * time.Millisecond
	time.Sleep(sleep)
	if e := tm.Elapsed(); e < sleep {
		t.Fatalf("Elapsed() = %v, want >= %v", e, sleep)
	}
}

func TestNewConfig(t *testing.T) {
	enabled := NewConfig(false)
	if enabled.Disabled {
		t.Error("NewConfig(false).Disabled = true, want false")
	}
	if enabled.RunID != RunID() {
		t.Errorf("NewConfig(false).RunID = %q, want the process RunID %q", enabled.RunID, RunID())
	}

	disabled := NewConfig(true)
	if !disabled.Disabled {
		t.Error("NewConfig(true).Disabled = false, want true")
	}
	if disabled.RunID != enabled.RunID {
		t.Errorf("Config.RunID differs across NewConfig calls: %q vs %q — run ID must be process-scoped",
			disabled.RunID, enabled.RunID)
	}
}

// topLevelJSONKeys marshals v and returns its top-level object as a key->raw map.
func topLevelJSONKeys(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal(%T) error: %v", v, err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}
	return fields
}

var requiredUsageFields = []string{
	"run_id", "sequence", "start_time", "duration", "os", "arch",
	"provider_version", "terraform_version", "resource_type", "operation",
	"changed_attributes", "error",
}

// Every required contract field must appear on the wire for BOTH a fully
// populated event and a zero-value / read-shaped event. The zero-value case is
// the load-bearing one: duration, sequence, and changed_attributes are exactly
// the fields that are legitimately zero/empty on a common read, so this guards
// against a future ",omitempty" (or a switch to pointer fields) silently
// dropping them from the payload for the most common events — a regression a
// fully-populated fixture cannot catch.
func TestUsage_RequiredFieldsAlwaysPresent(t *testing.T) {
	cases := map[string]Usage{
		"populated": {
			RunID:             "run-1",
			Sequence:          3,
			StartTime:         time.Unix(0, 0).UTC(),
			Duration:          42,
			OS:                "darwin",
			Arch:              "arm64",
			ProviderVersion:   "2.0.0",
			TerraformVersion:  "1.9.0",
			ResourceType:      "confluent_kafka_topic",
			Operation:         OperationCreate,
			ChangedAttributes: []string{"config"},
			Error:             false,
		},
		// Everything left at its zero value except operation: duration 0,
		// sequence 0, error false, nil changed attributes, empty strings.
		"zero-value read": {
			Operation: OperationRead,
		},
	}

	for name, u := range cases {
		fields := topLevelJSONKeys(t, u)
		for _, key := range requiredUsageFields {
			if _, ok := fields[key]; !ok {
				t.Errorf("[%s] Usage JSON is missing required field %q", name, key)
			}
		}
		if _, ok := fields["stack_frames"]; ok {
			t.Errorf("[%s] stack_frames must be omitted when empty", name)
		}
	}
}

// stack_frames is optional (panic path only): omitted when empty, present when
// populated.
func TestUsage_StackFramesOptional(t *testing.T) {
	empty := topLevelJSONKeys(t, Usage{Operation: OperationDelete})
	if _, ok := empty["stack_frames"]; ok {
		t.Error("stack_frames must be omitted when empty")
	}

	populated := topLevelJSONKeys(t, Usage{
		Operation:   OperationDelete,
		StackFrames: []string{"runtime/panic.go:884"},
	})
	if _, ok := populated["stack_frames"]; !ok {
		t.Error("stack_frames must be present when populated")
	}
}

// changed_attributes is a required, non-omitempty array. A nil slice serializes
// as JSON null and a non-nil slice as [] — pinned here so downstream callers
// (B3) know to pass a non-nil (possibly empty) slice for read events that
// change nothing, rather than emitting null.
func TestUsage_ChangedAttributesSerialization(t *testing.T) {
	nilFields := topLevelJSONKeys(t, Usage{Operation: OperationRead})
	if got := string(nilFields["changed_attributes"]); got != "null" {
		t.Errorf("nil ChangedAttributes serialized as %s, want null", got)
	}

	emptyFields := topLevelJSONKeys(t, Usage{Operation: OperationRead, ChangedAttributes: []string{}})
	if got := string(emptyFields["changed_attributes"]); got != "[]" {
		t.Errorf("empty ChangedAttributes serialized as %s, want []", got)
	}
}

// operation is a fixed, compile-time-known set; the wire values must be exactly
// these lowercase verbs so downstream span names (<resource_type>.<operation>)
// stay stable.
func TestOperationConstants(t *testing.T) {
	want := map[Operation]string{
		OperationCreate: "create",
		OperationRead:   "read",
		OperationUpdate: "update",
		OperationDelete: "delete",
		OperationImport: "import",
	}
	for op, s := range want {
		if string(op) != s {
			t.Errorf("operation constant = %q, want %q", string(op), s)
		}
	}
}
