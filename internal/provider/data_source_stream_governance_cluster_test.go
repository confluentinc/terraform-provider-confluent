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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceStreamGovernanceScenarioName = "confluent_stream_governance Data Source Lifecycle"
	streamGovernanceResourceLabel          = "essentials"
)

var fullStreamGovernanceDataSourceLabel = fmt.Sprintf("data.confluent_stream_governance_cluster.%s", streamGovernanceResourceLabel)

func TestAccDataSourceStreamGovernanceCluster(t *testing.T) {
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/read_provisioned_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(streamGovernanceClusterUrlPath)).
		InScenario(dataSourceStreamGovernanceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponse, _ := ioutil.ReadFile("../testdata/stream_governance_cluster/read_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/stream-governance/v2/clusters")).
		InScenario(dataSourceStreamGovernanceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(streamGovernanceClusterEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponse),
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
				Config: testAccCheckDataSourceStreamGovernanceClusterConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckStreamGovernanceClusterExists(fullStreamGovernanceDataSourceLabel),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramId, streamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramPackage, streamGovernanceClusterPackage),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), streamGovernanceClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), streamGovernanceClusterRegionId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramDisplayName, streamGovernanceClusterDisplayName),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramApiVersion, streamGovernanceClusterApiVersion),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramKind, streamGovernanceClusterKind),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramResourceName, streamGovernanceClusterResourceName),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramHttpEndpoint, streamGovernanceClusterHttpEndpoint),
				),
			},
			{
				Config: testAccCheckDataSourceStreamGovernanceClusterConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullStreamGovernanceDataSourceLabel),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramId, streamGovernanceClusterId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramPackage, streamGovernanceClusterPackage),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), streamGovernanceClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), streamGovernanceClusterRegionId),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramDisplayName, streamGovernanceClusterDisplayName),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramApiVersion, streamGovernanceClusterApiVersion),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramKind, streamGovernanceClusterKind),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramResourceName, streamGovernanceClusterResourceName),
					resource.TestCheckResourceAttr(fullStreamGovernanceDataSourceLabel, paramHttpEndpoint, streamGovernanceClusterHttpEndpoint),
				),
			},
		},
	})
}

func testAccCheckDataSourceStreamGovernanceClusterConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_stream_governance_cluster" "essentials" {
		id = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, streamGovernanceClusterId, streamGovernanceClusterEnvironmentId)
}

func testAccCheckDataSourceStreamGovernanceClusterConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_stream_governance_cluster" "essentials" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, streamGovernanceClusterDisplayName, streamGovernanceClusterEnvironmentId)
}
