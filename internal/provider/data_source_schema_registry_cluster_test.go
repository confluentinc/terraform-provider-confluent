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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceSchemaRegistryScenarioName = "confluent_schema_registry Data Source Lifecycle"
	dataSourceSchemaRegistryLabel        = "essentials"
)

const (
	schemaRegistryClusterHttpEndpoint                 = "https://psrc-y1111.us-west-2.aws.confluent.cloud"
	schemaRegistryClusterPrivateEndpoint              = "https://lsrc.us-west-2.aws.private.stag.cpdev.cloud"
	schemaRegistryClusterPrivateEndpointRegionalKey   = "key1"
	schemaRegistryClusterPrivateEndpointRegionalValue = "value1"
	schemaRegistryClusterCatalogEndpoint              = "https://psrc-y1113.us-west-2.aws.confluent.cloud"
	schemaRegistryClusterRegionId                     = "us-east4"
	schemaRegistryClusterId                           = "lsrc-755ogo"
	schemaRegistryClusterResourceName                 = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-1jrymj/schema-registry=lsrc-755ogo"
	schemaRegistryClusterApiVersion                   = "srcm/v3"
	schemaRegistryClusterKind                         = "Cluster"
	schemaRegistryClusterPackage                      = "ESSENTIALS"
	schemaRegistryClusterDisplayName                  = "Stream Governance Package"
	schemaRegistryClusterCloudType                    = "AWS"
)

var schemaRegistryClusterUrlPath = fmt.Sprintf("/srcm/v3/clusters/%s", schemaRegistryClusterId)

var fullSchemaRegistryDataSourceLabel = fmt.Sprintf("data.confluent_schema_registry_cluster.%s", dataSourceSchemaRegistryLabel)

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
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(dataSourceSchemaRegistryScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
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
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
				),
			},
			{
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
				),
			},
			{
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithJustEnvironmentSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpoint, schemaRegistryClusterHttpEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
				),
			},
		},
	})
}

func TestAccDataSourceSchemaRegistryClusterPrivate(t *testing.T) {
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

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_provisioned_cluster_private.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(schemaRegistryClusterUrlPath)).
		InScenario(dataSourceSchemaRegistryScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters_private.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(dataSourceSchemaRegistryScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
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
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpointPrivate, schemaRegistryClusterPrivateEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.%s", paramRestEndpointPrivateRegional, schemaRegistryClusterPrivateEndpointRegionalKey), schemaRegistryClusterPrivateEndpointRegionalValue),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
				),
			},
			{
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpointPrivate, schemaRegistryClusterPrivateEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.%s", paramRestEndpointPrivateRegional, schemaRegistryClusterPrivateEndpointRegionalKey), schemaRegistryClusterPrivateEndpointRegionalValue),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
				),
			},
			{
				Config: testAccCheckDataSourceSchemaRegistryClusterConfigWithJustEnvironmentSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClusterExists(fullSchemaRegistryDataSourceLabel),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramId, schemaRegistryClusterId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramPackage, schemaRegistryClusterPackage),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), testEnvironmentId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRegion, schemaRegistryClusterRegionId),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramDisplayName, schemaRegistryClusterDisplayName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramApiVersion, schemaRegistryClusterApiVersion),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramKind, schemaRegistryClusterKind),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramResourceName, schemaRegistryClusterResourceName),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramRestEndpointPrivate, schemaRegistryClusterPrivateEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, fmt.Sprintf("%s.%s", paramRestEndpointPrivateRegional, schemaRegistryClusterPrivateEndpointRegionalKey), schemaRegistryClusterPrivateEndpointRegionalValue),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCatalogEndpoint, schemaRegistryClusterCatalogEndpoint),
					resource.TestCheckResourceAttr(fullSchemaRegistryDataSourceLabel, paramCloud, schemaRegistryClusterCloudType),
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
	`, mockServerUrl, schemaRegistryClusterId, testEnvironmentId)
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
	`, mockServerUrl, schemaRegistryClusterDisplayName, testEnvironmentId)
}

func testAccCheckDataSourceSchemaRegistryClusterConfigWithJustEnvironmentSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_schema_registry_cluster" "essentials" {
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, testEnvironmentId)
}

func testAccCheckSchemaRegistryClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s Schema Registry Cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Schema Registry Cluster", n)
		}

		return nil
	}
}
