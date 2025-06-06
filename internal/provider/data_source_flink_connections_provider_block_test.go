// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceConnectionScenarioName = "confluent_flink_connection Data Source Lifecycle"
)

func TestAccDataSourceConnectionProviderBlock(t *testing.T) {
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

	readCreatedConnectionResponse, _ := ioutil.ReadFile("../testdata/flink_connection/read_connection.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkConnectionPath)).
		InScenario(dataSourceConnectionScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedConnectionResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readConnectionResponse, _ := ioutil.ReadFile("../testdata/flink_connection/read_connection_list.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(createFlinkConnectionPath)).
		InScenario(dataSourceConnectionScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readConnectionResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	flinkConnectionDataSourceLabel := "test"
	fullConnectionDataSourceLabel := fmt.Sprintf("data.confluent_flink_connection.%s", flinkConnectionDataSourceLabel)
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceConnectionProviderBlockConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(fullConnectionDataSourceLabel),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramDisplayName, flinkConnectionDisplayName),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramType, flinkConnectionType),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramEndpoint, flinkEndpoint),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramData, "string"),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramStatus, "READY"),
					resource.TestCheckResourceAttr(fullConnectionDataSourceLabel, paramStatusDetail, "Lookup failed: ai.openai.com"),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, paramRestEndpoint),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullConnectionDataSourceLabel, "credentials.0.secret"),
				),
			},
		},
	})
}

func testAccCheckDataSourceConnectionProviderBlockConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		flink_api_key         = "%s"
		flink_api_secret      = "%s"
		flink_rest_endpoint   = "%s"
		flink_principal_id    = "%s" 
		organization_id       = "%s"
		environment_id        = "%s"
		flink_compute_pool_id = "%s"
	}
	data "confluent_flink_connection" "test" {
		display_name = "%s"
	  	type = "%s"
	}
	`, mockServerUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkConnectionDisplayName, flinkConnectionType)
}
