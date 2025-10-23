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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	kafkaClustersDataSourceScenarioName = "confluent_kafka_clusters Data Source Lifecycle"
	kafkaClustersDataSourceLabel        = "test_kafka_clusters_data_source_label"
	kafkaClustersLastPageToken          = "dyJpZCI6InNhLTd5OXbyby"
)

func TestAccDataSourceKafkaClusters(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerUrl := wiremockContainer.URI
	// mockServerUrl := "http://localhost:8080"
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readClustersPageOneResponse, _ := ioutil.ReadFile("../testdata/kafka/read_kafkas_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters")).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listKafkaClustersPageSize))).
		//WithQueryParam("page_token", wiremock.Matching("^$")). // or Absent() if supported
		InScenario(kafkaClustersDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo("page_2").
		WillReturn(
			string(readClustersPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readClustersPageTwoResponse, _ := ioutil.ReadFile("../testdata/kafka/read_kafkas_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listKafkaClustersPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(kafkaClustersLastPageToken)).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		InScenario(kafkaClustersDataSourceScenarioName).
		WhenScenarioStateIs("page_2").
		WillSetStateTo(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullKafkaClustersDataSourceLabel := fmt.Sprintf("data.confluent_kafka_clusters.%s", kafkaClustersDataSourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceKafkaClusters(mockServerUrl, kafkaClustersDataSourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKafkaClustersExists(fullKafkaClustersDataSourceLabel),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, fmt.Sprintf("%s.#", paramClusters), "2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.id", "lkc-29ynpv"),
				),
			},
		},
	})
}

func testAccCheckDataSourceKafkaClusters(mockServerUrl, kafkaClustersDataSourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_kafka_clusters" "%s" {
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, kafkaClustersDataSourceLabel, testEnvironmentId)
}

func testAccCheckKafkaClustersExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s kafka cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s kafka cluster", n)
		}

		return nil
	}
}
