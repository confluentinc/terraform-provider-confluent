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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SchemaRegistryClusterCompatibilityLevelDataSourceScenarioName           = "confluent_schema_registry_cluster_config Data Source Lifecycle"
	testNumberOfSchemaRegistryClusterCompatibilityLevelDataSourceAttributes = 5
)

var fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel = fmt.Sprintf("data.confluent_schema_registry_cluster_config.%s", testSchemaResourceLabel)

func TestAccDataSchemaRegistryClusterCompatibilityLevelSchema(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockSchemaTestServerUrl = wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockSchemaTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedSchemaRegistryClusterCompatibilityLevelResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster_compatibility_level/read_created_schema_registry_cluster_compatibility_level.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(updateSchemaRegistryClusterCompatibilityLevelPath)).
		InScenario(SchemaRegistryClusterCompatibilityLevelDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSchemaRegistryClusterCompatibilityLevelResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckSchemaRegistryClusterCompatibilityLevelDataSourceConfig(confluentCloudBaseUrl, mockSchemaTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "schema_registry_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "schema_registry_cluster.0.id", testStreamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "rest_endpoint", mockSchemaTestServerUrl),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "credentials.0.key", testSchemaRegistryKey),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "credentials.0.secret", testSchemaRegistrySecret),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "compatibility_level", testSchemaRegistryClusterCompatibilityLevel),
					resource.TestCheckResourceAttr(fullSchemaRegistryClusterCompatibilityLevelDataSourceLabel, "%", strconv.Itoa(testNumberOfSchemaRegistryClusterCompatibilityLevelDataSourceAttributes)),
				),
			},
		},
	})
}

func testAccCheckSchemaRegistryClusterCompatibilityLevelDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_schema_registry_cluster_config" "%s" {
	  schema_registry_cluster {
        id = "%s"
      }
      rest_endpoint = "%s"
      credentials {
        key = "%s"
        secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, testSchemaResourceLabel, testStreamGovernanceClusterId, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret)
}
