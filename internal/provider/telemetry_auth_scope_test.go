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
	"testing"

	"github.com/confluentinc/terraform-provider-confluent/internal/provider/telemetry"
)

// TestTelemetryDisabledWithoutTopLevelIdentity covers TFCA-B7: a provider
// configured with no top-level Cloud identity (e.g. only resource-scoped Kafka
// credentials, which are never passed here) reports nothing — the runtime is
// disabled with no transport, so zero telemetry calls are made. Reporting with
// no identity is a no-op, never an error.
func TestTelemetryDisabledWithoutTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	// Default endpoint (would otherwise enable), but no cloud key and no OAuth/STS
	// token — mirroring a Kafka-only provider configuration.
	publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "", "", nil, nil, false)

	rt := publishedTelemetry.Load()
	if rt == nil {
		t.Fatal("expected a published runtime")
	}
	if !rt.config.Disabled {
		t.Errorf("runtime must be disabled when no top-level Cloud identity is configured")
	}
	if rt.reporter != nil {
		t.Errorf("no transport should be built without a top-level identity, got %T", rt.reporter)
	}

	// With no transport, the late-binding reporter is a no-op: it must not panic
	// and makes zero calls (there is nothing to send through).
	publishedTelemetryReporter{}.Report(telemetry.Usage{ResourceType: "confluent_kafka_topic", Operation: telemetry.OperationCreate})
}

// TestTelemetryEnabledWithTopLevelIdentity is the positive control: a top-level
// Cloud API key on the default endpoint yields an enabled runtime.
func TestTelemetryEnabledWithTopLevelIdentity(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "cloud-key", "cloud-secret", nil, nil, false)
	rt := publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled || rt.reporter == nil {
		t.Fatalf("expected an enabled runtime with a top-level identity, got %+v", rt)
	}
	if c, ok := rt.reporter.(interface{ Close() }); ok {
		c.Close()
	}
}
