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

package telemetry

import (
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// A provider subprocess handles many concurrent CRUD/import calls over its life
// but has a single identity throughout. These process-scoped primitives supply
// that identity: one run ID and one monotonic sequence counter, shared by every
// operation the process handles. Together they let downstream consumers
// reconstruct the invocation order of Terraform's concurrent, independently
// timed operations, which is not otherwise preserved.
var (
	runIDOnce sync.Once
	runID     string
	sequence  atomic.Int64
)

// RunID returns the process-scoped run identifier, generating it exactly once
// on first use and returning that same value for the life of the process.
//
// The sync.Once — rather than minting the ID inside the single
// ConfigureProvider call — is deliberate defense-in-depth. Terraform Core calls
// ConfigureProvider exactly once per provider subprocess today, but that is an
// implementation detail of Core, not a documented protocol guarantee, so RunID
// does not rely on it.
//
// See Usage.RunID for the (non-)correlation semantics callers must respect.
func RunID() string {
	runIDOnce.Do(func() {
		runID = uuid.New().String()
	})
	return runID
}

// NextSequence returns the next sequence number for this process, starting at 1
// and increasing by one per call. It is safe for concurrent use by Terraform's
// parallel graph walk.
//
// Call it at wrapper entry, not at report time, so the number reflects the
// order operations were invoked in even when the matching report is later
// dropped. Because reports can be dropped, the emitted stream of sequence
// numbers within a run may contain gaps; consumers must not assume every value
// arrived.
func NextSequence() int64 {
	return sequence.Add(1)
}
