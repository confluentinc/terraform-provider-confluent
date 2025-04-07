// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	connectArtifactDataSourceScenarioName = "confluent_connect_artifact Data Source Lifecycle"
	connectArtifactDataSourceLabel        = "example"
)

var fullConnectArtifactDataSourceLabel = fmt.Sprintf("data.confluent_connect_artifact.%s", connectArtifactDataSourceLabel)

func TestAccConnectArtifactDataSource(t *testing.T) {
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
	createConnectArtifactResponse, _ := json.Marshal(map[string]interface{}{
		"id": connectArtifactId,
		"spec": map[string]interface{}{
			"display_name":   connectArtifactUniqueName,
			"cloud":          connectArtifactCloud,
			"region":         connectArtifactRegion,
			"environment":    connectArtifactEnvironmentId,
			"content_format": connectArtifactContentFormat,
			"description":    connectArtifactDescription,
		},
		"status": map[string]interface{}{
			"phase": "PROVISIONED",
		},
	})

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/cam/v1/connect-artifacts/%s", connectArtifactId))).
		InScenario(connectArtifactDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createConnectArtifactResponse),
			map[string]string{"Content-Type": "application/json"},
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectArtifactDataSourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramId, connectArtifactId),
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramDisplayName, connectArtifactUniqueName),
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramCloud, connectArtifactCloud),
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramRegion, connectArtifactRegion),
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramContentFormat, connectArtifactContentFormat),
					resource.TestCheckResourceAttr(fullConnectArtifactDataSourceLabel, paramDescription, connectArtifactDescription),
				),
			},
		},
	})
}

func testAccCheckConnectArtifactDataSourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_connect_artifact" "%s" {
	  id = "%s"
	  environment {
		id = "%s"
	  }
	  cloud  = "%s"
	  region = "%s"
	}
	`, mockServerUrl, connectArtifactDataSourceLabel, connectArtifactId, connectArtifactEnvironmentId, connectArtifactCloud, connectArtifactRegion)
}
