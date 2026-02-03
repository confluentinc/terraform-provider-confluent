// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	endpointApiVersion             = "endpoint/v1"
	endpointDataSourceScenarioName = "confluent_endpoint Data Source Lifecycle"
	endpointKind                   = "Endpoint"
	endpointResourceLabel          = "test_endpoint"
	testEndpointEnvironmentId      = "env-abc123"
	testEndpointServiceKafka       = "KAFKA"
	testEndpointServiceSchemaReg   = "SCHEMA_REGISTRY"
	testEndpointCloud              = "AWS"
	testEndpointRegion             = "us-west-2"
	testEndpointResourceId         = "lkc-abc123"
)

func TestAccDataSourceEndpointKafka(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEndpointDataSourceLabel := fmt.Sprintf("data.confluent_endpoint.%s", endpointResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEndpointKafka(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "2"),

					// First endpoint (REST)
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.api_version", endpointApiVersion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.kind", endpointKind),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.service", testEndpointServiceKafka),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.cloud", testEndpointCloud),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.region", testEndpointRegion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.endpoint_type", "REST"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.connection_type", "PRIVATE_LINK"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.is_private", "true"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.environment.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.environment.0.id", testEndpointEnvironmentId),

					// Second endpoint (BOOTSTRAP)
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.api_version", endpointApiVersion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.kind", endpointKind),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.service", testEndpointServiceKafka),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.endpoint_type", "BOOTSTRAP"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.connection_type", "PRIVATE_LINK"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointKafkaWithFilters(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_filtered.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("cloud", wiremock.EqualTo(testEndpointCloud)).
		WithQueryParam("region", wiremock.EqualTo(testEndpointRegion)).
		WithQueryParam("is_private", wiremock.EqualTo("true")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEndpointDataSourceLabel := fmt.Sprintf("data.confluent_endpoint.%s", endpointResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEndpointKafkaWithFilters(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.cloud", testEndpointCloud),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.region", testEndpointRegion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.is_private", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointSchemaRegistry(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_schema_registry_endpoints.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceSchemaReg)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEndpointDataSourceLabel := fmt.Sprintf("data.confluent_endpoint.%s", endpointResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEndpointSchemaRegistry(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.service", testEndpointServiceSchemaReg),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.endpoint_type", "REST"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointWithResourceFilter(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_with_resource.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("resource", wiremock.EqualTo(testEndpointResourceId)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEndpointDataSourceLabel := fmt.Sprintf("data.confluent_endpoint.%s", endpointResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEndpointWithResource(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.service", testEndpointServiceKafka),
					// Verify that the returned endpoint has the resource we filtered by
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.0.id", testEndpointResourceId),
				),
			},
		},
	})
}

func testAccCheckDataSourceEndpointKafka(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka)
}

func testAccCheckDataSourceEndpointKafkaWithFilters(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "%s"
			cloud = "%s"
			region = "%s"
			is_private = true
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka, testEndpointCloud, testEndpointRegion)
}

func testAccCheckDataSourceEndpointSchemaRegistry(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceSchemaReg)
}

func testAccCheckDataSourceEndpointWithResource(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = "%s"
			}
			service = "%s"
			resource = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka, testEndpointResourceId)
}

func testAccCheckEndpointExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s endpoint has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s endpoint", resourceName)
		}

		return nil
	}
}
