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
	"testing"
	"time"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// capturingPoster is a telemetry.Poster that forwards each delivered Usage to a
// channel, so a test can observe what actually reached the transport's sender.
type capturingPoster struct {
	got chan telemetry.Usage
}

func (c capturingPoster) Post(_ context.Context, u telemetry.Usage) error {
	c.got <- u
	return nil
}

// TestTelemetryEnabledPathEmitsEndToEnd exercises the whole enabled chain that no
// per-component test composes: a resource from New()'s real, wrapped ResourcesMap
// -> the reporter captured at New() (publishedTelemetryReporter) -> an enabled
// published runtime -> the real bounded-worker Transport -> a poster that
// receives the Usage.
//
// It is the regression guard for the New() wiring line itself: if
// `reporter: publishedTelemetryReporter{}` were ever reverted to a no-op, or the
// published-runtime lookup broke, no other test would fail — this one would time
// out. Invoking Read with a nil meta panics inside the real resource; the B4
// recovery converts that to a reported crash event, which is exactly the Usage we
// assert arrives (proving the chain end to end without any network).
func TestTelemetryEnabledPathEmitsEndToEnd(t *testing.T) {
	restorePublishedTelemetry(t)

	poster := capturingPoster{got: make(chan telemetry.Usage, 4)}
	transport := telemetry.NewTransport(poster, context.Background())
	defer transport.Close()

	// Publish an ENABLED runtime backed by the real Transport (no endpoint/network
	// involved — the poster is a local capture).
	publishedTelemetry.Store(&telemetryRuntime{
		config:   telemetry.NewConfig(false),
		reporter: transport,
	})

	// Drive a real, New()-wrapped resource. This asserts the New() wiring, not a
	// synthetic fixture.
	p := New(testVersion, "")()
	r, ok := p.ResourcesMap["confluent_environment"]
	if !ok || r.ReadContext == nil {
		t.Fatal("confluent_environment with a ReadContext is required for this test")
	}
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
		t.Fatal("no telemetry reached the poster: the New()->published reporter->transport->poster chain is broken")
	}
}
