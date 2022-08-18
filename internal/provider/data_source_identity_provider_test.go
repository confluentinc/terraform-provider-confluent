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
	dataSourceIdentityProviderScenarioName = "confluent_identity_provider Data Source Lifecycle"
	identityProviderDataSourceLabel        = "example"
)

var fullIdentityProviderDataSourceLabel = fmt.Sprintf("data.confluent_identity_provider.%s", identityProviderDataSourceLabel)

func TestAccDataSourceIdentityProvider(t *testing.T) {
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

	readCreatedIdentityProviderResponse, _ := ioutil.ReadFile("../testdata/identity_provider/read_created_identity_provider.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(dataSourceIdentityProviderScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedIdentityProviderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readIdentityProvidersResponse, _ := ioutil.ReadFile("../testdata/identity_provider/read_identity_providers.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/identity-providers")).
		InScenario(dataSourceIdentityProviderScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readIdentityProvidersResponse),
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
				Config: testAccCheckDataSourceIdentityProviderWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderExists(fullIdentityProviderDataSourceLabel),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramId, identityProviderId),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramDisplayName, identityProviderDisplayName),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramDescription, identityProviderDescription),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramIssuer, identityProviderIssuer),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramJwksUri, identityProviderJwksUri),
				),
			},
			{
				Config: testAccCheckDataSourceIdentityProviderWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderExists(fullIdentityProviderDataSourceLabel),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramId, identityProviderId),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramDisplayName, identityProviderDisplayName),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramDescription, identityProviderDescription),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramIssuer, identityProviderIssuer),
					resource.TestCheckResourceAttr(fullIdentityProviderDataSourceLabel, paramJwksUri, identityProviderJwksUri),
				),
			},
		},
	})
}

func testAccCheckDataSourceIdentityProviderWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_identity_provider" "%s" {
		display_name = "%s"
	}
	`, mockServerUrl, identityProviderDataSourceLabel, identityProviderDisplayName)
}

func testAccCheckDataSourceIdentityProviderWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_identity_provider" "%s" {
	    id = "%s"
	}
	`, mockServerUrl, identityProviderDataSourceLabel, identityProviderId)
}
