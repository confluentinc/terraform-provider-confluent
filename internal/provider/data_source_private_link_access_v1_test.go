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
	dataSourcePrivateLinkAccessScenarioName = "confluent_private_link_access_v1 Data Source Lifecycle"
	plaDataSourceLabel                      = "example"
	plaDataSourceDisplayName                = "prod-pl-use2"
)

var fullPrivateLinkAccessDataSourceLabel = fmt.Sprintf("data.confluent_private_link_access_v1.%s", plaDataSourceLabel)

func TestAccDataSourcePrivateLinkAccess(t *testing.T) {
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

	readCreatedAwsPrivateLinkAccessResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_created_pla.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(awsPlaUrlPath)).
		InScenario(dataSourcePrivateLinkAccessScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedAwsPrivateLinkAccessResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPlasResponse, _ := ioutil.ReadFile("../testdata/private_link_access/aws/read_plas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/networking/v1/private-link-accesses")).
		InScenario(dataSourcePrivateLinkAccessScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(awsPlaEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readPlasResponse),
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
				Config: testAccCheckDataSourcePlaWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPlaExists(fullPrivateLinkAccessDataSourceLabel),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "id", awsPlaId),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "display_name", plaDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "environment.0.id", awsPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "network.0.id", awsPlaNetworkId),
				),
			},
			{
				Config: testAccCheckDataSourcePlaWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsPlaExists(fullPrivateLinkAccessDataSourceLabel),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "id", awsPlaId),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "display_name", plaDataSourceDisplayName),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "aws.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "aws.0.account", awsAccountNumber),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "azure.#", "0"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "environment.0.id", awsPlaEnvironmentId),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "network.#", "1"),
					resource.TestCheckResourceAttr(fullPrivateLinkAccessDataSourceLabel, "network.0.id", awsPlaNetworkId),
				),
			},
		},
	})
}

func testAccCheckDataSourcePlaWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_private_link_access_v1" "%s" {
		display_name = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, plaDataSourceLabel, plaDataSourceDisplayName, awsPlaEnvironmentId)
}

func testAccCheckDataSourcePlaWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_private_link_access_v1" "%s" {
	    id = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, plaDataSourceLabel, awsPlaId, awsPlaEnvironmentId)
}
