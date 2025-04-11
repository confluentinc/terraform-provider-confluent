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
	dataSourceComputePoolScenarioName = "confluent_flink_compute_pool Data Source Lifecycle"
)

var fullComputePoolDataSourceLabel = fmt.Sprintf("data.confluent_flink_compute_pool.%s", networkDataSourceLabel)

func TestAccDataSourceComputePool(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	readCreatedAwsComputePoolResponse, _ := ioutil.ReadFile("../testdata/compute_pool/read_created_compute_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/compute-pools/lfcp-abc123")).
		InScenario(dataSourceComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsComputePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readComputePoolsResponse, _ := ioutil.ReadFile("../testdata/compute_pool/read_compute_pools.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/compute-pools")).
		InScenario(dataSourceComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readComputePoolsResponse),
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
				Config: testAccCheckDataSourceAwsComputePoolConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolExists(fullComputePoolDataSourceLabel),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramId, flinkComputePoolId),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramDisplayName, flinkComputePoolDisplayName),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramCloud, flinkComputePoolCloud),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramRegion, flinkComputePoolRegion),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramMaxCfu, strconv.Itoa(flinkComputePoolDefaultMaxCfu)),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkComputePoolEnvironmentId),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramApiVersion, flinkComputePoolApiVersion),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramKind, flinkComputePoolKind),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramResourceName, flinkComputePoolResourceName),
				),
			},
			{
				Config: testAccCheckDataSourceAzureComputePoolConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolExists(fullComputePoolDataSourceLabel),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramId, flinkComputePoolId),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramDisplayName, flinkComputePoolDisplayName),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramCloud, flinkComputePoolCloud),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramRegion, flinkComputePoolRegion),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramMaxCfu, strconv.Itoa(flinkComputePoolDefaultMaxCfu)),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkComputePoolEnvironmentId),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramApiVersion, flinkComputePoolApiVersion),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramKind, flinkComputePoolKind),
					resource.TestCheckResourceAttr(fullComputePoolDataSourceLabel, paramResourceName, flinkComputePoolResourceName),
				),
			},
		},
	})
}

func testAccCheckDataSourceAzureComputePoolConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_flink_compute_pool" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, networkDataSourceLabel, flinkComputePoolDisplayName, flinkComputePoolEnvironmentId)
}

func testAccCheckDataSourceAwsComputePoolConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_flink_compute_pool" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, networkDataSourceLabel, flinkComputePoolId, flinkComputePoolEnvironmentId)
}
