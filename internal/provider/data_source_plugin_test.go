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

// TestAccDataSourcePlugin covers the confluent_plugin data source, which resolves a plugin by id
// within an environment. It reuses the resource's own fixture rather than a synthetic one, so the
// values asserted here are the same ones TestAccPlugin asserts against a real API shape.
func TestAccDataSourcePlugin(t *testing.T) {
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

	readPluginResponse, _ := os.ReadFile("../testdata/plugin/read_created_plugin.json")
	readPluginStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/ccpm/v1/plugins/%s", pluginId))).
		WithQueryParam("environment", wiremock.EqualTo(pluginEnvironment)).
		InScenario(pluginDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(readPluginStub)

	fullPluginDataSourceLabel := "data.confluent_plugin.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourcePluginConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "id", pluginId),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "display_name", "plugin-name"),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "description", "plugin-description"),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "cloud", "AWS"),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "runtime_language", "JAVA"),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "environment.0.id", pluginEnvironment),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "api_version", "ccpm/v1"),
					resource.TestCheckResourceAttr(fullPluginDataSourceLabel, "kind", "CustomConnectPlugin"),
				),
			},
		},
	})

	// The environment is a query parameter rather than part of the path, so a read that dropped it
	// would still hit /ccpm/v1/plugins/{id} and pass every assertion above. Only the stub's
	// WithQueryParam makes that a miss, and only counting the stub turns the miss into a failure.
	readRequests, err := wiremockClient.GetCountRequests(readPluginStub.Request())
	if err != nil {
		t.Fatalf("could not count GET /ccpm/v1/plugins/%s requests: %s", pluginId, err)
	}
	if readRequests == 0 {
		t.Fatalf("expected at least one environment-scoped GET /ccpm/v1/plugins/%s request, but found none", pluginId)
	}
}

func testAccCheckDataSourcePluginConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_plugin" "main" {
		id = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, pluginId, pluginEnvironment)
}
