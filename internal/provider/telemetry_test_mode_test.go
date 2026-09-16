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
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const telemetryTestModeScenario = "telemetryTestModeScenarioName"

// TestAccTelemetryDisabledDuringAcceptanceTests is the end-to-end guard that no
// telemetry escapes during a testacc run: a full environment create/destroy
// lifecycle under TF_ACC emits zero telemetry calls, verified with checkStubCount
// against a stubbed terraform-usage endpoint (a non-zero create-stub count proves
// the lifecycle really ran). It does not isolate the test-mode gate on its own —
// an acceptance run also uses a non-default (mock) endpoint, which disables
// reporting too; the gate itself is isolated by TestPublishTelemetryRuntime's
// test-mode subtest. This test guards the composed outcome.
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

	// Sanity: the full lifecycle really ran (create and destroy each happened once)...
	checkStubCount(t, wiremockClient, createEnvStub, "POST /org/v2/environments", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteEnvStub, "DELETE /org/v2/environments/env-1jrymj", expectedCountOne)
	// ...and yet zero telemetry was emitted during the acceptance run.
	checkStubCount(t, wiremockClient, telemetryStub, "POST /terraform-usage/v1/usages", expectedCountZero)
}
