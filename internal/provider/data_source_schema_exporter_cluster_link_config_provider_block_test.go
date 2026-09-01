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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

// Exercises Option #2 (Schema Registry metadata set in the provider block): the data source
// reads credentials/endpoint/cluster from the provider block, so the inline write-back branch
// is skipped and schema_registry_cluster/credentials/rest_endpoint stay unset.
func TestAccDataSourceSchemaExporterClusterLinkConfigWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	clusterLinkConfigResponse, _ := ioutil.ReadFile("../testdata/schema_exporter/cluster_link_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readSchemaExporterClusterLinkConfigUrlPath)).
		InScenario(schemaExporterClusterLinkConfigDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(clusterLinkConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSchemaExporterClusterLinkConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaExists(schemaExporterClusterLinkConfigDataSourceLabel),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "id", fmt.Sprintf("%s/exporter1", testStreamGovernanceClusterId)),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "name", "exporter1"),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "schema_registry_cluster.#", "0"),
					resource.TestCheckNoResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "schema_registry_cluster.0.id"),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "rest_endpoint"),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, fmt.Sprintf("%s.%%", paramConfigs), "1"),
					resource.TestCheckResourceAttr(schemaExporterClusterLinkConfigDataSourceLabel, "config.topic.config.sync.associations.filters", testSchemaExporterClusterLinkConfigValue),
				),
			},
		},
	})
}

func testAccCheckDataSourceSchemaExporterClusterLinkConfigWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint                      = "%s"
	  schema_registry_rest_endpoint = "%s"
	  schema_registry_api_key       = "%s"
	  schema_registry_api_secret    = "%s"
	  schema_registry_id            = "%s"
	}
	data "confluent_schema_exporter_cluster_link_config" "main" {
	  name = "exporter1"
	}
	`, confluentCloudBaseUrl, mockServerUrl, testSchemaRegistryKey, testSchemaRegistrySecret, testStreamGovernanceClusterId)
}
