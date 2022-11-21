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
	dataSourceSchemaRegistryScenarioName = "confluent_schema_registry Data Source Lifecycle"
	schemaRegistryResourceLabel          = "essentials"
)

var fullSchemaRegistryDataSourceLabel = fmt.Sprintf("data.confluent_schema_registry_cluster.%s", schemaRegistryResourceLabel)

func TestAccDataSourceSchemaRegistryCluster(t *testing.T) {
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioned_cluster.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(dataSourceSchemaRegistryScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v2/clusters")).
		InScenario(dataSourceSchemaRegistryScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(schemaRegistryClusterEnvironmentId)).
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
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSchemaRegistryClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), schemaRegistryClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
				),
			},
			{
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), schemaRegistryClusterEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramRegion), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramRegion, paramId), schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
				),
			},
		},
	})
}

func testAccCheckDataSourceSchemaRegistryClusterConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_schema_registry_cluster" "essentials" {
		id = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, schemaRegistryClusterId, schemaRegistryClusterEnvironmentId)
}

func testAccCheckDataSourceSchemaRegistryClusterConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_schema_registry_cluster" "essentials" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, schemaRegistryClusterDisplayName, schemaRegistryClusterEnvironmentId)
}
