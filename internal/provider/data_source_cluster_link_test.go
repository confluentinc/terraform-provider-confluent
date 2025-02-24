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
	clusterLinkDataSourceLabel              = "test_cluster_link_data_source_label"
	numberOfClusterLinkDataSourceAttributes = "8"
)

var fullClusterLinkDataSourceLabel = fmt.Sprintf("data.confluent_cluster_link.%s", clusterLinkDataSourceLabel)

func TestAccDataSourceClusterLink(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockClusterLinkTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockClusterLinkTestServerUrl)

	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedClusterLinkResponse, _ := ioutil.ReadFile("../testdata/cluster_link/read_created_cluster_link_destination.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterLinkResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedClusterLinkConfigResponse, _ := ioutil.ReadFile("../testdata/cluster_link/read_created_cluster_link_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readClusterLinkSourceOutboundConfigPath)).
		InScenario(clusterLinkScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterLinkConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckClusterLinkDataSourceConfig(confluentCloudBaseUrl, mockClusterLinkTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterLinkExists(fullClusterLinkDataSourceLabel),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "link_name", clusterLinkName),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "rest_endpoint", mockClusterLinkTestServerUrl),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "kafka_cluster.0.%", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "kafka_cluster.0.id", sourceClusterId),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "credentials.#", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "credentials.0.%", "2"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "credentials.0.key", sourceClusterApiKey),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "credentials.0.secret", sourceClusterApiSecret),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "cluster_link_id", "qz0HDEV-Qz2B5aPFpcWQJQ"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "id", fmt.Sprintf("%s/%s", sourceClusterId, clusterLinkName)),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "link_state", "ACTIVE"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "config.%", "1"),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, fmt.Sprintf("config.%s", firstClusterClusterLinkConfigName), firstClusterClusterLinkConfigValue),
					resource.TestCheckResourceAttr(fullClusterLinkDataSourceLabel, "%", numberOfClusterLinkDataSourceAttributes),
				),
			},
		},
	})
}

func testAccCheckClusterLinkDataSourceConfig(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_cluster_link" "%s" {
	  link_name = "%s"
      rest_endpoint = "%s"
	  kafka_cluster {
        id = "%s"
      }
      credentials {
		key = "%s"
		secret = "%s"
	  }
	}
	`, confluentCloudBaseUrl, clusterLinkDataSourceLabel,
		clusterLinkName, mockServerUrl, sourceClusterId, sourceClusterApiKey, sourceClusterApiSecret)
}
