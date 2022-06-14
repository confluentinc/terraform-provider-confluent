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
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourcePeeringScenarioName = "confluent_peering_v1 Data Source Lifecycle"
	peeringDataSourceLabel        = "example"
	peeringDataSourceDisplayName  = "my-test-peering"
)

var fullPeeringDataSourceLabel = fmt.Sprintf("data.confluent_peering_v1.%s", peeringDataSourceLabel)

func TestAccDataSourcePeering(t *testing.T) {
	containerPort := "8080"
	containerPortTcp := fmt.Sprintf("%s/tcp", containerPort)
	ctx := context.Background()
	listeningPort := wait.ForListeningPort(nat.Port(containerPortTcp))
	req := testcontainers.ContainerRequest{
		Image:        "rodolpheche/wiremock",
		ExposedPorts: []string{containerPortTcp},
		WaitingFor:   listeningPort,
	}
	wiremockContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})

	require.NoError(t, err)

	// nolint:errcheck
	defer wiremockContainer.Terminate(ctx)

	host, err := wiremockContainer.Host(ctx)
	require.NoError(t, err)

	wiremockHttpMappedPort, err := wiremockContainer.MappedPort(ctx, nat.Port(containerPort))
	require.NoError(t, err)

	mockServerUrl := fmt.Sprintf("http://%s:%s", host, wiremockHttpMappedPort.Port())
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedAwsPeeringResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_created_peering.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPeeringUrlPath)).
		InScenario(dataSourcePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsPeeringResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPeeringsResponse, _ := ioutil.ReadFile("../testdata/peering/aws/read_peerings.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/peerings")).
		InScenario(dataSourcePeeringScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPeeringEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPeeringsResponse),
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
				Config: testAccCheckDataSourcePeeringWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullPeeringDataSourceLabel),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "display_name", peeringDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
			{
				Config: testAccCheckDataSourcePeeringWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPeeringExists(fullPeeringDataSourceLabel),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "id", awsPeeringId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "display_name", peeringDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "gcp.#", "0"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "environment.0.id", awsPeeringEnvironmentId),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPeeringDataSourceLabel, "network.0.id", awsPeeringNetworkId),
				),
			},
		},
	})
}

func testAccCheckDataSourcePeeringWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_peering_v1" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, peeringDataSourceLabel, peeringDataSourceDisplayName, awsPeeringEnvironmentId)
}

func testAccCheckDataSourcePeeringWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_peering_v1" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, peeringDataSourceLabel, awsPeeringId, awsPeeringEnvironmentId)
}
