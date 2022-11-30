// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"
)

const (
	scenarioStateKsqlHasBeenCreated   = "A new ksqlDB cluster has been just created"
	scenarioStateKsqlHasBeenUpdated   = "The new ksqlDB cluster's kind has been just updated"
	scenarioStateKsqlHasBeenDeleted   = "The new ksqlDB cluster has been deleted"
	ksqlScenarioName                  = "confluent_ksql_cluster Resource Lifecycle"
	ksqlResourceLabel                 = "basic-cluster"
	containerPort                     = "8080"
	ksqlId                            = "lksql-0000"
	ksqlDataSourceDisplayName         = "ksqlDB_cluster_0"
	ksqlApiVersion                    = "ksqldbcm/v2"
	ksqlKind                          = "Cluster"
	ksqlCsuTest1                      = "4"
	ksqlCsuTest2                      = "8"
	ksqlCredentialIdentity            = "u-a83k9b"
	ksqlDisplayName                   = "ksqlDB_cluster_0"
	ksqlUseDetailedProcessingLogTest1 = "true"
	ksqlUseDetailedProcessingLogTest2 = "false"
)

var fullKsqlResourceLabel = fmt.Sprintf("confluent_ksql_cluster.%s", ksqlResourceLabel)

var createKsqlPath = "/ksqldbcm/v2/clusters"
var readKsqlPath = fmt.Sprintf(createKsqlPath+"/%s", ksqlId)

var resourceCommonChecks = resource.ComposeTestCheckFunc(
	testAccCheckKsqlClusterExists(fullKsqlResourceLabel),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramApiVersion, ksqlApiVersion),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramKind, ksqlKind),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramId, ksqlId),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramTopicPrefix, "pksqlc-00000"),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramDisplayName, ksqlDataSourceDisplayName),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramRestEndpoint, "https://pksqlc-00000.us-central1.gcp.glb.confluent.cloud"),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramResourceName, "crn://confluent.cloud/organization=1111aaaa-11aa-11aa-11aa-111111aaaaaa/environment=env-abc123/cloud-cluster=lkc-00000/ksql=ksqlDB_cluster_1"),
	resource.TestCheckNoResourceAttr(fullKsqlResourceLabel, paramHttpEndpoint),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramStorage, "125"),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, "environment.0.id", kafkaEnvId),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, "kafka_cluster.0.id", kafkaClusterId),
	resource.TestCheckResourceAttr(fullKsqlResourceLabel, "credential_identity.0.id", ksqlCredentialIdentity))

func TestAccCreateKsqlClusterError(t *testing.T) {

	ctx := context.Background()

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/501_internal_server_error.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusNotImplemented,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      ksqlResourceConfig(mockServerUrl, ksqlCsuTest1, ksqlUseDetailedProcessingLogTest1),
				ExpectError: regexp.MustCompile("error creating ksqlDB Cluster"),
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterStub, fmt.Sprintf("POST %s", createKsqlPath), 1)
}

func TestAccImportKsqlCluster(t *testing.T) {

	ctx := context.Background()

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONING_ksql_4_csu.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONED_ksql_4_csu.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = createDefaultDeleteStub(wiremockClient)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: ksqlResourceConfig(mockServerUrl, ksqlCsuTest1, ksqlUseDetailedProcessingLogTest1),
				Check:  testAccCheckKsqlClusterExists(fullKsqlResourceLabel),
			},
			{
				ResourceName: fullKsqlResourceLabel,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					clusterId := resources[fullKsqlResourceLabel].Primary.ID
					environmentId := resources[fullKsqlResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + clusterId, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccReadKsqlClusterError(t *testing.T) {

	ctx := context.Background()

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)

	require.NoError(t, err)
	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONED_ksql_4_csu.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/501_internal_server_error.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusNotImplemented, //blocks retry
		))

	_ = createDefaultDeleteStub(wiremockClient)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      ksqlResourceConfig(mockServerUrl, ksqlCsuTest1, ksqlUseDetailedProcessingLogTest1),
				ExpectError: regexp.MustCompile("error waiting for ksqlDB Cluster"),
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterStub, fmt.Sprintf("POST %s", createKsqlPath), 1)
}

func TestAccKsqlCluster(t *testing.T) {

	ctx := context.Background()

	wiremockContainer, err := createWiremockContainer(ctx, containerPort)
	require.NoError(t, err)

	wiremockClient, mockServerUrl, err := createWiremockClient(ctx, wiremockContainer, containerPort)
	require.NoError(t, err)

	defer cleanUp(ctx, wiremockContainer, wiremockClient)

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONING_ksql_4_csu.json")
	createClusterStub := wiremock.Post(wiremock.URLPathEqualTo(createKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(createClusterResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	readCreatedClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONED_ksql_4_csu.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenCreated).
		WillReturn(
			string(readCreatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteClusterStubGeneral := wiremock.Delete(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenCreated).
		WillSetStateTo(scenarioStateKsqlHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStubGeneral)

	createClusterUpdatedResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONING_ksql_8_csu.json")
	createClusterStub = wiremock.Post(wiremock.URLPathEqualTo(createKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenDeleted).
		WillSetStateTo(scenarioStateKsqlHasBeenUpdated).
		WillReturn(
			string(createClusterUpdatedResponse),
			contentTypeJSONHeader,
			http.StatusAccepted,
		)
	_ = wiremockClient.StubFor(createClusterStub)

	readUpdatedClusterResponse, _ := ioutil.ReadFile("../testdata/ksql/PROVISIONED_ksql_8_csu.json")
	updateClusterStub := wiremock.Patch(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateClusterStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenUpdated).
		WillReturn(
			string(readUpdatedClusterResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteClusterStub := wiremock.Delete(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenUpdated).
		WillSetStateTo(scenarioStateKsqlHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteClusterStub)

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/ksql/403_forbidden.json")

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readKsqlPath)).
		InScenario(ksqlScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WhenScenarioStateIs(scenarioStateKsqlHasBeenDeleted).
		WillReturn(
			string(readDeletedEnvResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckKsqlClusterDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: ksqlResourceConfig(mockServerUrl, ksqlCsuTest1, ksqlUseDetailedProcessingLogTest1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKsqlClusterExists(fullKsqlResourceLabel),
					resourceCommonChecks,
					resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramCsu, ksqlCsuTest1),
					resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramUseDetailedProcessingLog, ksqlUseDetailedProcessingLogTest1),
				),
			},
			{
				Config: ksqlResourceConfig(mockServerUrl, ksqlCsuTest2, ksqlUseDetailedProcessingLogTest2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckKsqlClusterExists(fullKsqlResourceLabel),
					resourceCommonChecks,
					resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramCsu, ksqlCsuTest2),
					resource.TestCheckResourceAttr(fullKsqlResourceLabel, paramUseDetailedProcessingLog, ksqlUseDetailedProcessingLogTest2),
				),
			},
		},
	})

	checkStubCount(t, wiremockClient, createClusterStub, fmt.Sprintf("POST %s", createKsqlPath), 2)
	checkStubCount(t, wiremockClient, deleteClusterStub, fmt.Sprintf("DELETE %s", readKsqlPath), 2)
}

func testAccCheckKsqlClusterDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each environment is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_ksql_cluster" {
			continue
		}
		deletedClusterId := rs.Primary.ID
		req := c.ksqlClient.ClustersKsqldbcmV2Api.GetKsqldbcmV2Cluster(c.ksqlApiContext(context.Background()), deletedClusterId).Environment(kafkaEnvId)
		deletedCluster, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// /ksqldbcm/v2/clusters/{nonExistentClusterId/deletedClusterID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the environment is destroyed.
			return nil
		} else if err == nil && deletedCluster.Id != nil {
			// Otherwise return the error
			if *deletedCluster.Id == rs.Primary.ID {
				return fmt.Errorf("ksqlDB cluster (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

// simple delete stub for when no scenario or change of state is needed
func createDefaultDeleteStub(client *wiremock.Client) error {
	err := client.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readKsqlPath)).
		WithQueryParam("environment", wiremock.EqualTo(kafkaEnvId)).
		WillReturn("", contentTypeJSONHeader, http.StatusNoContent))
	if err != nil {
		return err
	}
	return nil
}

func ksqlResourceConfig(mockServerUrl, csu, useDetailedProcessingLog string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_ksql_cluster" "%s" {
		display_name = "%s"
		use_detailed_processing_log = %s
		csu = %s
		kafka_cluster {
			id = "%s"
		}
		credential_identity {
			id = "%s"
		}
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, ksqlResourceLabel, ksqlDisplayName, useDetailedProcessingLog, csu, kafkaClusterId, ksqlCredentialIdentity, kafkaEnvId)
}

func testAccCheckKsqlClusterExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s ksqlDB cluster has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s ksqlDB cluster", n)
		}

		return nil
	}
}
