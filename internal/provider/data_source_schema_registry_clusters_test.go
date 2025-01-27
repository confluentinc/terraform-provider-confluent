// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SRClustersDataSourceScenarioName = "confluent_schema_registry_clusters Data Source Lifecycle"
)

var environments = []string{"env-1jnw8z", "env-7n1r31"}

func TestAccDataSourceSchemaRegistryClusters(t *testing.T) {
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

	readEnvironmentsPageOneResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_envs_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEnvironmentsPageSize))).
		InScenario(SRClustersDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readEnvironmentsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponseOne, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters_1jnw8z.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(SRClustersDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-1jnw8z")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponseOne),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponseTwo, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters_7n1r31.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(SRClustersDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-7n1r31")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponseTwo),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullSRClustersDataSourceLabel := fmt.Sprintf("data.confluent_schema_registry_clusters.%s", "main")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSRClusters(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSRClustersExists(fullSRClustersDataSourceLabel),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.#", paramClusters), "2"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.id", paramClusters), "lsrc-755ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.environment.0.id", paramClusters), "env-1jnw8z"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.resource_name", paramClusters), "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-1jnw8z/schema-registry=lsrc-755ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.display_name", paramClusters), "Stream Governance Package"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.rest_endpoint", paramClusters), "https://psrc-y1111.us-west-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.package", paramClusters), "ESSENTIALS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.region", paramClusters), "us-east4"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.cloud", paramClusters), "AWS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.id", paramClusters), "lsrc-756ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.environment.0.id", paramClusters), "env-7n1r31"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.resource_name", paramClusters), "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-7n1r31/schema-registry=lsrc-756ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.display_name", paramClusters), "Stream Governance Package"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.rest_endpoint", paramClusters), "https://psrc-y1111.us-west-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.catalog_endpoint", paramClusters), "https://psrc-y1113.us-west-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.package", paramClusters), "ESSENTIALS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.region", paramClusters), "us-east4"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.cloud", paramClusters), "AWS"),
				),
			},
		},
	})
}

func TestAccDataSourceSchemaRegistryClustersPrivate(t *testing.T) {
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

	readEnvironmentsPageOneResponse, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_envs_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEnvironmentsPageSize))).
		InScenario(SRClustersDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readEnvironmentsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponseOne, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters_private_1jnw8z.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(SRClustersDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-1jnw8z")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponseOne),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersResponseTwo, _ := ioutil.ReadFile("../testdata/schema_registry_cluster/read_clusters_private_7n1r31.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/srcm/v3/clusters")).
		InScenario(SRClustersDataSourceScenarioName).
		WithQueryParam("environment", wiremock.EqualTo("env-7n1r31")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponseTwo),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullSRClustersDataSourceLabel := fmt.Sprintf("data.confluent_schema_registry_clusters.%s", "main")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceSRClusters(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSRClustersExists(fullSRClustersDataSourceLabel),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.#", paramClusters), "2"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.id", paramClusters), "lsrc-755ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.environment.0.id", paramClusters), "env-1jnw8z"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.resource_name", paramClusters), "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-1jnw8z/schema-registry=lsrc-755ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.display_name", paramClusters), "Stream Governance Package"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.package", paramClusters), "ESSENTIALS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.region", paramClusters), "us-east4"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.0.cloud", paramClusters), "AWS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.id", paramClusters), "lsrc-756ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.environment.0.id", paramClusters), "env-7n1r31"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.resource_name", paramClusters), "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-7n1r31/schema-registry=lsrc-756ogo"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.display_name", paramClusters), "Stream Governance Package"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.private_rest_endpoint", paramClusters), "https://lsrc.us-west-2.aws.private.stag.cpdev.cloud"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.private_regional_rest_endpoint.key1", paramClusters), "value1"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.catalog_endpoint", paramClusters), "https://psrc-y1113.us-west-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.package", paramClusters), "ESSENTIALS"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.region", paramClusters), "us-east4"),
					resource.TestCheckResourceAttr(fullSRClustersDataSourceLabel, fmt.Sprintf("%s.1.cloud", paramClusters), "AWS"),
				),
			},
		},
	})
}

func testAccCheckDataSourceSRClusters(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_schema_registry_clusters" "main" {
	}
	`, mockServerUrl)
}

func testAccCheckSRClustersExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s SR Cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s SR Cluster", n)
		}

		return nil
	}
}
