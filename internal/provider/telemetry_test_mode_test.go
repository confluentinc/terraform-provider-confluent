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
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const telemetryTestModeScenario = "telemetryTestModeScenarioName"

// TestTelemetryDisabledInTestMode isolates the TFCA-B8 gate: even on the default
// endpoint and with a top-level identity (a config that would otherwise report),
// test mode disables reporting so no transport is built.
func TestTelemetryDisabledInTestMode(t *testing.T) {
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "")

	publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "key", "secret", nil, nil, true /* testMode */)
	rt := publishedTelemetry.Load()
	if rt == nil || !rt.config.Disabled || rt.reporter != nil {
		t.Fatalf("test mode must disable telemetry with no transport, got %+v", rt)
	}

	// Control: the same inputs with testMode=false do enable it (proving the only
	// difference is the gate). Close the transport it stands up.
	publishTelemetryRuntime(t.Context(), defaultCloudEndpoint, "ua", "key", "secret", nil, nil, false)
	rt = publishedTelemetry.Load()
	if rt == nil || rt.config.Disabled || rt.reporter == nil {
		t.Fatalf("expected telemetry enabled when not in test mode, got %+v", rt)
	}
	if c, ok := rt.reporter.(interface{ Close() }); ok {
		c.Close()
	}
}

// TestAccTelemetryDisabledDuringAcceptanceTests is an end-to-end "no telemetry
// escapes during a testacc run" guard: a full resource lifecycle under TF_ACC
// emits zero telemetry calls, verified with checkStubCount against a stubbed
// terraform-usage endpoint (with a non-zero create-stub count proving the
// lifecycle really ran). It does NOT isolate the test-mode gate specifically —
// an acceptance run structurally uses a non-default (mock) endpoint, which also
// disables reporting — so the gate is isolated by the unit test
// TestTelemetryDisabledInTestMode; this test guards the composed outcome.
func TestAccTelemetryDisabledDuringAcceptanceTests(t *testing.T) {
	restorePublishedTelemetry(t)
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()
	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	// A stub for the telemetry endpoint. If anything reported, this counter would
	// be non-zero; the test asserts it stays at zero.
	telemetryStub := wiremock.Post(wiremock.URLPathEqualTo("/terraform-usage/v1/usages")).
		WillReturn("", contentTypeJSONHeader, http.StatusOK)
	_ = wiremockClient.StubFor(telemetryStub)

	// A minimal create -> destroy environment lifecycle, so real CRUD runs.
	createEnvResponse, _ := os.ReadFile("../testdata/environment/create_env.json")
	createEnvStub := wiremock.Post(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(telemetryTestModeScenario).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateEnvHasBeenCreated).
		WillReturn(string(createEnvResponse), contentTypeJSONHeader, http.StatusCreated)
	_ = wiremockClient.StubFor(createEnvStub)

	readCreatedEnvResponse, _ := os.ReadFile("../testdata/environment/read_created_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments/env-1jrymj")).
		InScenario(telemetryTestModeScenario).
		WhenScenarioStateIs(scenarioStateEnvHasBeenCreated).
		WillReturn(string(readCreatedEnvResponse), contentTypeJSONHeader, http.StatusOK))

	deleteEnvStub := wiremock.Delete(wiremock.URLPathEqualTo("/org/v2/environments/env-1jrymj")).
		InScenario(telemetryTestModeScenario).
		WhenScenarioStateIs(scenarioStateEnvHasBeenCreated).
		WillSetStateTo(scenarioStateEnvHasBeenDeleted).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent)
	_ = wiremockClient.StubFor(deleteEnvStub)

	readDeletedEnvResponse, _ := os.ReadFile("../testdata/environment/read_deleted_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments/env-1jrymj")).
		InScenario(telemetryTestModeScenario).
		WhenScenarioStateIs(scenarioStateEnvHasBeenDeleted).
		WillReturn(string(readDeletedEnvResponse), contentTypeJSONHeader, http.StatusNotFound))

	environmentResourceLabel := "telemetry_test_env"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				// display_name must match the value in the create/read fixtures, else
				// the post-apply plan is non-empty and the step fails.
				Config: testAccCheckEnvironmentConfig(mockServerUrl, environmentResourceLabel, "test_env_display_name", "ESSENTIALS"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel)),
				),
			},
		},
	})

	// Sanity: the lifecycle really ran (create happened once)...
	checkStubCount(t, wiremockClient, createEnvStub, "POST /org/v2/environments", expectedCountOne)
	// ...and yet zero telemetry was emitted during the acceptance run.
	checkStubCount(t, wiremockClient, telemetryStub, "POST /terraform-usage/v1/usages", expectedCountZero)
}
