// Copyright 2022  Confluent Inc. All Rights Reserved.
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
	"github.com/stretchr/testify/require"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	ksqlScenarioDataSourceName = "confluent_ksql_cluster Data Source Lifecycle"
	ksqlDataSourceLabel        = "test_ksql_data_source_label"
)

var fullKsqlDataSourceLabel = fmt.Sprintf("data.confluent_ksql_cluster.%s", ksqlDataSourceLabel)

var datasourceCommonChecks = resource.ComposeTestCheckFunc(
	testAccCheckEnvironmentExists(fullKsqlDataSourceLabel),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramApiVersion, ksqlApiVersion),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramKind, ksqlKind),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramId, ksqlId),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramDisplayName, ksqlDataSourceDisplayName),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramCsu, "4"),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramStorage, "125"),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramUseDetailedProcessingLog, "true"),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramTopicPrefix, "pksqlc-00000"),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, paramHttpEndpoint, "https://pksqlc-00000.us-central1.gcp.glb.confluent.cloud"),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, "environment.0.id", kafkaEnvId),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, "kafka_cluster.0.id", kafkaClusterId),
	resource.TestCheckResourceAttr(fullKsqlDataSourceLabel, "credential_identity.0.id", ksqlCredentialIdentity),
)

func TestAccDataSourceWithIdKsql(t *testing.T) {

	ctx := context.Background()
	containerPort := "8080"

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	provisioningKsqlCluster, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONED_ksql_4_csu.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/ksqldbcm/v2/clusters/%s", ksqlId))).
		InScenario(ksqlScenarioDataSourceName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(provisioningKsqlCluster),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	provisionedKsqlCluster, _ := ioutil.ReadFile("../testdata/ksql/ksql_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ksqldbcm/v2/clusters")).
		InScenario(ksqlScenarioDataSourceName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(provisionedKsqlCluster),
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
				Config: datasourceConfigDisplayName(mockServerUrl),
				Check:  datasourceCommonChecks,
			},
		},
	})
}

func TestAccDataSourceListKsql(t *testing.T) {

	ctx := context.Background()
	containerPort := "8080"

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	provisionedKsqlCluster, _ := ioutil.ReadFile("../testdata/ksql/ksql_clusters.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ksqldbcm/v2/clusters")).
		InScenario(ksqlScenarioDataSourceName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(provisionedKsqlCluster),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: datasourceConfigDisplayName(mockServerUrl),
				Check:  datasourceCommonChecks,
			},
		},
	})
}

func TestAccDataSourceKsqlApi5xxError(t *testing.T) {

	ctx := context.Background()
	containerPort := "8080"

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	errorResponse, _ := ioutil.ReadFile("../testdata/ksql/501_internal_server_error.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/ksqldbcm/v2/clusters/%s", ksqlId))).
		InScenario(ksqlScenarioDataSourceName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(errorResponse),
			contentTypeJSONHeader,
			//501 is the only status code without retries, otherwise tests will take 10+ seconds
			http.StatusNotImplemented,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      datasourceConfigId(mockServerUrl),
				ExpectError: regexp.MustCompile("error reading ksqlDB cluster"),
			},
		},
	})
}

func TestAccDataSourceKsqlApi4xxError(t *testing.T) {

	ctx := context.Background()
	containerPort := "8080"

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	errorResponse, _ := ioutil.ReadFile("../testdata/ksql/401_Unathorized.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/ksqldbcm/v2/clusters/%s", ksqlId))).
		InScenario(ksqlScenarioDataSourceName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(errorResponse),
			contentTypeJSONHeader,
			http.StatusUnauthorized,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      datasourceConfigId(mockServerUrl),
				ExpectError: regexp.MustCompile("Valid authentication credentials must be provided"),
			},
		},
	})
}

func datasourceConfigId(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_ksql_cluster" "%s" {
		id = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, ksqlDataSourceLabel, ksqlId, kafkaEnvId)
}

func datasourceConfigDisplayName(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_ksql_cluster" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, ksqlDataSourceLabel, ksqlDisplayName, kafkaEnvId)
}
