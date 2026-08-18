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
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
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
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.0.id", testEndpointResourceId),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.0.kind", "Cluster"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.gateway.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.access_point.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.access_point.0.id", "ap-abc123"),

					// Second endpoint (BOOTSTRAP) - has no resource, gateway, or access_point
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.api_version", endpointApiVersion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.kind", endpointKind),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.service", testEndpointServiceKafka),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.endpoint_type", "BOOTSTRAP"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.connection_type", "PRIVATE_LINK"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.resource.#", "0"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.gateway.#", "0"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.access_point.#", "0"),
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

func TestAccDataSourceEndpointFlink(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_flink_endpoints.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceFlink)).
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
				Config: testAccCheckDataSourceEndpointFlink(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.service", testEndpointServiceFlink),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.endpoint_type", "LANGUAGE_SERVICE"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.connection_type", "PUBLIC"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.is_private", "false"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointIsPrivateFalse(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_public.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("is_private", wiremock.EqualTo("false")).
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
				Config: testAccCheckDataSourceEndpointIsPrivateFalse(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.is_private", "false"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.connection_type", "PUBLIC"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointCloudOnly(t *testing.T) {
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
				Config: testAccCheckDataSourceEndpointCloudOnly(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.cloud", testEndpointCloud),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointRegionOnly(t *testing.T) {
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
		WithQueryParam("region", wiremock.EqualTo(testEndpointRegion)).
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
				Config: testAccCheckDataSourceEndpointRegionOnly(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.region", testEndpointRegion),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointEmptyResult(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_empty.json")
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
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "0"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointPagination(t *testing.T) {
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

	readEndpointsPageOneResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEndpointsPageTwoResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(testEndpointPageToken)).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsPageTwoResponse),
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
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.id", "ep-page1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.id", "ep-page2"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointReadError(t *testing.T) {
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

	errorResponse, _ := os.ReadFile("../testdata/endpoint/501_internal_server_error.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(errorResponse),
			contentTypeJSONHeader,
			// 501 is the only status code without retries, otherwise tests will take 10+ seconds
			http.StatusNotImplemented,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckDataSourceEndpointKafka(mockServerUrl, endpointResourceLabel),
				ExpectError: regexp.MustCompile("error reading endpoints"),
			},
		},
	})
}

func TestAccDataSourceEndpointPartialFields(t *testing.T) {
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

	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_partial_fields.json")
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

					// First endpoint has no environment block at all
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.id", "ep-noenv"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.environment.#", "0"),

					// Second endpoint has a resource block with no kind
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.id", "ep-nokind"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.resource.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.resource.0.id", "lkc-def456"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.resource.0.kind", ""),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointAllFiltersCombined(t *testing.T) {
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
		WithQueryParam("cloud", wiremock.EqualTo(testEndpointCloud)).
		WithQueryParam("region", wiremock.EqualTo(testEndpointRegion)).
		WithQueryParam("is_private", wiremock.EqualTo("true")).
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
				Config: testAccCheckDataSourceEndpointAllFiltersCombined(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.cloud", testEndpointCloud),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.region", testEndpointRegion),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.is_private", "true"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.resource.0.id", testEndpointResourceId),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointNextEmptyString(t *testing.T) {
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

	// Only one stub is registered: if the provider incorrectly treated an explicit empty
	// "next" as a page to follow, the second request would hit an unstubbed URL and fail.
	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_next_empty.json")
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
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "1"),
				),
			},
		},
	})
}

func TestAccDataSourceEndpointEmptyEnvironmentId(t *testing.T) {
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

	// No stub is registered: environment.id = "" must fail before any HTTP request is made
	// (extractStringValueFromBlock uses GetOk, which can't distinguish "unset" from "set to
	// the zero value", so an explicit empty string collapses to the same "" as if the
	// Required block were missing entirely). A request hitting this unstubbed URL would fail
	// with a different, unstubbed-request error rather than matching ExpectError below.

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckDataSourceEndpointEmptyEnvironmentId(mockServerUrl, endpointResourceLabel),
				ExpectError: regexp.MustCompile("environment ID is required in filter"),
			},
		},
	})
}

func TestAccDataSourceEndpointNextMalformedURL(t *testing.T) {
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

	// Only one stub is registered: "next" is a non-empty URL missing a page_token query
	// param, so extractPageToken must fail loudly rather than the loop silently treating it
	// as the last page or looping on an unstubbed request.
	readEndpointsResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_next_malformed.json")
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

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckDataSourceEndpointKafka(mockServerUrl, endpointResourceLabel),
				ExpectError: regexp.MustCompile("could not parse the value"),
			},
		},
	})
}

func TestAccDataSourceEndpointPaginationWithFilter(t *testing.T) {
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

	// Both stubs require the "cloud" filter, confirming loadEndpoints carries optional
	// filters through every page of the loop, not just the first request.
	readEndpointsPageOneResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("cloud", wiremock.EqualTo(testEndpointCloud)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEndpointsPageTwoResponse, _ := os.ReadFile("../testdata/endpoint/read_kafka_endpoints_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/endpoint/v1/endpoints")).
		WithQueryParam("environment", wiremock.EqualTo(testEndpointEnvironmentId)).
		WithQueryParam("service", wiremock.EqualTo(testEndpointServiceKafka)).
		WithQueryParam("cloud", wiremock.EqualTo(testEndpointCloud)).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEndpointsPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(testEndpointPageToken)).
		InScenario(endpointDataSourceScenarioName).
		WillReturn(
			string(readEndpointsPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEndpointDataSourceLabel := fmt.Sprintf("data.confluent_endpoint.%s", endpointResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEndpointCloudOnly(mockServerUrl, endpointResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEndpointExists(fullEndpointDataSourceLabel),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.#", "2"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.0.id", "ep-page1"),
					resource.TestCheckResourceAttr(fullEndpointDataSourceLabel, "endpoints.1.id", "ep-page2"),
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

func testAccCheckDataSourceEndpointFlink(mockServerUrl, label string) string {
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
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceFlink)
}

func testAccCheckDataSourceEndpointIsPrivateFalse(mockServerUrl, label string) string {
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
			is_private = false
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka)
}

func testAccCheckDataSourceEndpointCloudOnly(mockServerUrl, label string) string {
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
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka, testEndpointCloud)
}

func testAccCheckDataSourceEndpointRegionOnly(mockServerUrl, label string) string {
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
			region = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka, testEndpointRegion)
}

func testAccCheckDataSourceEndpointAllFiltersCombined(mockServerUrl, label string) string {
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
			resource = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointEnvironmentId, testEndpointServiceKafka, testEndpointCloud, testEndpointRegion, testEndpointResourceId)
}

func testAccCheckDataSourceEndpointEmptyEnvironmentId(mockServerUrl, label string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_endpoint" "%s" {
		filter {
			environment {
				id = ""
			}
			service = "%s"
		}
	}
	`, mockServerUrl, label, testEndpointServiceKafka)
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
