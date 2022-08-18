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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateIdentityProviderHasBeenCreated             = "The new identity_provider has been just created"
	scenarioStateIdentityProviderDescriptionHaveBeenUpdated = "The new identity_provider's description have been just updated"
	scenarioStateIdentityProviderHasBeenDeleted             = "The new identity_provider has been deleted"
	identityProviderScenarioName                            = "confluent_identity_provider Resource Lifecycle"
	identityProviderId                                      = "op-4EY"
	identityProviderDisplayName                             = "My OIDC Provider"
	identityProviderDescription                             = "fake description"
	identityProviderIssuer                                  = "https://login.microsoftonline.com/11111111-0000-0000-0000-b3d3d184f1a5/v2.0"
	identityProviderJwksUri                                 = "https://login.microsoftonline.com/common/discovery/v2.0/keys"
)

func TestAccIdentityProvider(t *testing.T) {
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
	createSaResponse, _ := ioutil.ReadFile("../testdata/identity_provider/create_identity_provider.json")
	createSaStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/identity-providers")).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIdentityProviderHasBeenCreated).
		WillReturn(
			string(createSaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createSaStub)

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/identity_provider/read_created_identity_provider.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityProviderHasBeenCreated).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedIdentityProviderResponse, _ := ioutil.ReadFile("../testdata/identity_provider/read_updated_identity_provider.json")
	patchSaStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityProviderHasBeenCreated).
		WillSetStateTo(scenarioStateIdentityProviderDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedIdentityProviderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchSaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityProviderDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedIdentityProviderResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedSaResponse, _ := ioutil.ReadFile("../testdata/identity_provider/read_deleted_identity_provider.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityProviderHasBeenDeleted).
		WillReturn(
			string(readDeletedSaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteSaStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s", identityProviderId))).
		InScenario(identityProviderScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityProviderDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateIdentityProviderHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSaStub)

	// in order to test tf update (step #3)
	identityProviderUpdatedDescription := "fake description updated"
	identityProviderUpdatedDisplayName := "My OIDC Provider updated"
	identityProviderResourceLabel := "test_identity_provider_resource_label"
	fullIdentityProviderResourceLabel := fmt.Sprintf("confluent_identity_provider.%s", identityProviderResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIdentityProviderDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIdentityProviderConfig(mockServerUrl, identityProviderResourceLabel, identityProviderDisplayName, identityProviderDescription, identityProviderIssuer, identityProviderJwksUri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderExists(fullIdentityProviderResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramId, identityProviderId),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramDisplayName, identityProviderDisplayName),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramDescription, identityProviderDescription),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramIssuer, identityProviderIssuer),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramJwksUri, identityProviderJwksUri),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullIdentityProviderResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckIdentityProviderConfig(mockServerUrl, identityProviderResourceLabel, identityProviderUpdatedDisplayName, identityProviderUpdatedDescription, identityProviderIssuer, identityProviderJwksUri),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityProviderExists(fullIdentityProviderResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramId, identityProviderId),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramDisplayName, identityProviderUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramDescription, identityProviderUpdatedDescription),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramIssuer, identityProviderIssuer),
					resource.TestCheckResourceAttr(fullIdentityProviderResourceLabel, paramJwksUri, identityProviderJwksUri),
				),
			},
			{
				ResourceName:      fullIdentityProviderResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createSaStub, "POST /iam/v2/identity-providers", expectedCountOne)
	checkStubCount(t, wiremockClient, patchSaStub, "PATCH /iam/v2/identity-providers/op-537", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteSaStub, "DELETE /iam/v2/identity-providers/op-537", expectedCountOne)
}

func testAccCheckIdentityProviderDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each service account is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_identity_provider" {
			continue
		}
		deletedIdentityProviderId := rs.Primary.ID
		req := c.oidcClient.IdentityProvidersIamV2Api.GetIamV2IdentityProvider(c.oidcApiContext(context.Background()), deletedIdentityProviderId)
		deletedIdentityProvider, response, err := req.Execute()
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(response)
		if isResourceNotFound {
			return nil
		} else if err == nil && deletedIdentityProvider.Id != nil {
			// Otherwise return the error
			if *deletedIdentityProvider.Id == rs.Primary.ID {
				return fmt.Errorf("service account (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckIdentityProviderConfig(mockServerUrl, identityProviderResourceLabel, identityProviderDisplayName, identityProviderDescription, identityProviderIssuer, identityProviderJwksUri string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_identity_provider" "%s" {
		display_name = "%s"
		description = "%s"
		issuer      = "%s"
		jwks_uri    = "%s"
	}
	`, mockServerUrl, identityProviderResourceLabel, identityProviderDisplayName, identityProviderDescription, identityProviderIssuer, identityProviderJwksUri)
}

func testAccCheckIdentityProviderExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s identity provider has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s identity provider", n)
		}

		return nil
	}
}
