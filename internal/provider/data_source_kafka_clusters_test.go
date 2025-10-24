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

	testKafkaHttpEndpoint2      = "https://pkc-3w22w.us-central1.gcp.confluent.cloud:443"
	testKafkaBootstrapEndpoint2 = "SASL_SSL://pkc-3w22w.us-central1.gcp.confluent.cloud:9092"
	testKafkaDisplayName2       = "TestCluster #2"
	testKafkaRbacCrn2           = "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-1jrymj/cloud-cluster=lkc-29ynpv"
)

func TestAccDataSourceKafkaClusters(t *testing.T) {
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

	readClustersPageOneResponse, _ := ioutil.ReadFile("../testdata/kafka/read_kafkas_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters")).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listKafkaClustersPageSize))).
		InScenario(kafkaClustersDataSourceScenarioName).
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
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.basic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.basic.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.enterprise.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.freight.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.dedicated.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.byok.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.endpoints.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.network.0.id", kafkaNetworkId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.rbac_crn", kafkaRbacCrn),

					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.id", "lkc-29ynpv"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.display_name", testKafkaDisplayName2),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.bootstrap_endpoint", testKafkaBootstrapEndpoint2),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.basic.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.basic.0.%", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.standard.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.enterprise.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.freight.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.dedicated.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.byok.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.endpoints.#", "0"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.environment.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.environment.0.id", testEnvironmentId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.network.#", "1"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.network.0.id", kafkaNetworkId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.rest_endpoint", testKafkaHttpEndpoint2),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.1.rbac_crn", testKafkaRbacCrn2),
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
