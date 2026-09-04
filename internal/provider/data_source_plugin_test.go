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

// TestAccDataSourcePlugin exercises both lookup branches of the confluent_plugin data source:
// by id (GET /ccpm/v1/plugins/{id}) and by display_name, which has no server-side filter and so
// paginates the collection and scans it client-side.
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
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/ccpm/v1/plugins/%s", pluginId))).
		WithQueryParam("environment", wiremock.EqualTo(pluginEnvironment)).
		InScenario(pluginDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	listPluginsResponse, _ := os.ReadFile("../testdata/plugin/list_plugins.json")
	listPluginsStub := wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins")).
		WithQueryParam("environment", wiremock.EqualTo(pluginEnvironment)).
		WithQueryParam("page_size", wiremock.EqualTo("99")).
		InScenario(pluginDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(listPluginsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(listPluginsStub)

	fullPluginDataSourceLabel := "data.confluent_plugin.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourcePluginConfigUsingId(mockServerUrl),
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
			{
				Config: testAccCheckDataSourcePluginConfigUsingDisplayName(mockServerUrl),
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

	// Both steps assert the same attribute values -- deliberately, since both lookups resolve the
	// same plugin -- so an identical set of Check funcs cannot by itself prove the second step took
	// the display_name branch rather than re-reading by id. This does: the by-id branch never calls
	// the collection endpoint, so a non-zero count is only reachable through
	// pluginDataSourceReadUsingDisplayName. The exact count is Terraform's plan/refresh/apply
	// cadence and is not worth pinning.
	listRequests, err := wiremockClient.GetCountRequests(listPluginsStub.Request())
	if err != nil {
		t.Fatalf("could not count GET /ccpm/v1/plugins requests: %s", err)
	}
	if listRequests == 0 {
		t.Fatalf("expected at least one GET /ccpm/v1/plugins request, so the display_name lookup is actually exercised, but found none")
	}
}

func testAccCheckDataSourcePluginConfigUsingId(mockServerUrl string) string {
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

func testAccCheckDataSourcePluginConfigUsingDisplayName(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_plugin" "main" {
		display_name = "plugin-name"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, pluginEnvironment)
}
