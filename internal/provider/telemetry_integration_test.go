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

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// capturingPoster forwards each delivered Usage to a channel for the test to observe.
type capturingPoster struct {
	got chan telemetry.Usage
}

func (c capturingPoster) Post(_ context.Context, u telemetry.Usage) error {
	c.got <- u
	return nil
}

// TestTelemetryEnabledPathEmitsEndToEnd checks the enabled path end to end: a
// New()-wrapped resource reports a Usage that travels through the published
// reporter and the real Transport to the poster. It guards the New() reporter
// wiring, which no other test exercises through the real transport.
func TestTelemetryEnabledPathEmitsEndToEnd(t *testing.T) {
	restorePublishedTelemetry(t)

	poster := capturingPoster{got: make(chan telemetry.Usage, 4)}
	transport := telemetry.NewTransport(poster, context.Background())
	defer transport.Close()

	// Enabled runtime backed by the real transport; the poster captures locally.
	publishedTelemetry.Store(&telemetryRuntime{
		config:   telemetry.NewConfig(false),
		reporter: transport,
	})

	// Drive a real New()-wrapped resource so this covers the actual wiring.
	p := New(testVersion, "")()
	r, ok := p.ResourcesMap["confluent_environment"]
	if !ok || r.ReadContext == nil {
		t.Fatal("confluent_environment with a ReadContext is required for this test")
	}
	// nil args make the read fail fast without a network call; the wrapper still
	// reports a Usage.
	_ = r.ReadContext(context.Background(), nil, nil)

	select {
	case u := <-poster.got:
		if u.ResourceType != "confluent_environment" {
			t.Errorf("ResourceType = %q, want confluent_environment", u.ResourceType)
		}
		if u.Operation != telemetry.OperationRead {
			t.Errorf("Operation = %q, want READ", u.Operation)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no telemetry reached the poster: the enabled reporter/transport chain is broken")
	}
}
