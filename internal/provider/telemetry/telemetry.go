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

// Package telemetry holds the client-side analytics data model and the
// process-scoped primitives used to correlate provider operations.
//
// It is intentionally self-contained: it owns only the payload struct
// (Usage), the process run identifier and sequence counter (run.go), the
// captured configuration snapshot (Config), and timing capture (Timer). The
// central CRUD/import wrapping that populates a Usage, the opt-out wiring that
// sets Config.Disabled, and the network transport that reports a Usage all
// live outside this package and consume it. It does no network or filesystem
// I/O, reads no environment variables, and does not reference the provider's
// Client struct.
package telemetry

import "time"

// Operation identifies which CRUD or import entry point produced a Usage event.
// The set is closed and known at compile time; these are the only valid values.
type Operation string

const (
	OperationCreate Operation = "create"
	OperationRead   Operation = "read"
	OperationUpdate Operation = "update"
	OperationDelete Operation = "delete"
	OperationImport Operation = "import"
)

// Usage is the metadata-only analytics payload emitted for a single resource
// CRUD or import operation. Its field names mirror the TFCA-A1 `terraform/v1`
// usage contract; the concrete wire client is generated separately (TFCA-A2),
// and this struct is the provider-internal model that populates it.
//
// By construction Usage never carries attribute values or the keys of a
// user-controlled map: ChangedAttributes holds schema-declared attribute names
// only (enforced by the TFCA-B2 build-time allowlist), and StackFrames is
// populated only on the panic path.
type Usage struct {
	// RunID is the process-scoped run identifier, stable for the life of one
	// provider subprocess (see RunID).
	//
	// It is process-scoped — not plan-scoped and not config-scoped. Terraform
	// Core launches a fresh provider subprocess for each phase and for each
	// aliased provider block, so:
	//   - A `plan` and its paired `apply` run as two separate processes and
	//     therefore carry two different, non-joinable RunIDs. RunID does not
	//     correlate a plan to its apply, even within a single `terraform apply`
	//     invocation.
	//   - Each aliased `provider "confluent"` instance runs in its own process
	//     and produces its own RunID. RunID does not correlate one aliased
	//     instance to another.
	// Downstream consumers must not assume RunID joins any of these.
	RunID string `json:"run_id"`

	// Sequence is the monotonic, per-process invocation order of this
	// operation, assigned at wrapper entry (see NextSequence). Because reports
	// can be dropped, the sequence values that actually arrive within a run may
	// contain gaps; consumers must not assume every value was emitted.
	Sequence int64 `json:"sequence"`

	// StartTime is when the operation began, captured at wrapper entry.
	StartTime time.Time `json:"start_time"`

	// Duration is the operation's elapsed time in milliseconds, measured with a
	// monotonic clock from entry to just before the wrapped function returns.
	Duration int64 `json:"duration"`

	// OS is the reporting host's operating system (runtime.GOOS), e.g. "darwin".
	OS string `json:"os"`

	// Arch is the reporting host's architecture (runtime.GOARCH), e.g. "arm64".
	Arch string `json:"arch"`

	// ProviderVersion is the released provider version, constant for the life
	// of the process.
	ProviderVersion string `json:"provider_version"`

	// TerraformVersion is the version of Terraform Core that launched the
	// provider.
	TerraformVersion string `json:"terraform_version"`

	// ResourceType is the resource's Terraform type, e.g. "confluent_kafka_topic".
	ResourceType string `json:"resource_type"`

	// Operation is which CRUD or import entry point ran.
	Operation Operation `json:"operation"`

	// ChangedAttributes lists schema-declared attribute names only — never a
	// map's keys and never any attribute value. Callers should supply a
	// non-nil slice (empty when nothing changed) so it serializes as [] rather
	// than null.
	ChangedAttributes []string `json:"changed_attributes"`

	// Error reports whether the operation returned an error or panicked.
	Error bool `json:"error"`

	// StackFrames is a trimmed stack trace, populated only on the panic path
	// and omitted otherwise.
	StackFrames []string `json:"stack_frames,omitempty"`
}

// Config is the process-scoped telemetry configuration captured once during
// provider configuration and only read thereafter.
//
// It is a value type holding only immutable scalar fields, so copying it shares
// no mutable state with the original — deliberately unlike the provider's
// Client struct, parts of which are mutated after configuration without a lock
// (so "the Client is safe to read concurrently" is not an assumption these
// fields can rely on). Value semantics alone do not publish it safely: the
// happens-before edge that lets the CRUD goroutines read it comes from writing
// the one canonical Config during provider configuration, before those
// goroutines start (a downstream, TFCA-B6/B3 concern).
type Config struct {
	// RunID is a snapshot of the process run identifier (see RunID).
	RunID string

	// Disabled reports whether telemetry reporting is turned off for this
	// process (opt-out environment variable or non-default endpoint). It is
	// populated by the opt-out wiring (TFCA-B6); the zero value leaves
	// telemetry enabled.
	Disabled bool
}

// NewConfig snapshots the process run identifier together with the opt-out
// decision into an immutable Config.
func NewConfig(disabled bool) Config {
	return Config{
		RunID:    RunID(),
		Disabled: disabled,
	}
}

// Timer captures the start of an operation so the wrapper can record the
// Usage.StartTime and Usage.Duration, timed from wrapper entry to just before
// the wrapped function returns.
type Timer struct {
	start time.Time
}

// StartTimer captures the current time as an operation's start. Call it at
// wrapper entry.
func StartTimer() Timer {
	return Timer{start: time.Now()}
}

// StartTime returns the instant the timer was started.
func (t Timer) StartTime() time.Time {
	return t.start
}

// Elapsed returns the time elapsed since the timer was started. Call it just
// before the wrapped function returns.
func (t Timer) Elapsed() time.Duration {
	return time.Since(t.start)
}
