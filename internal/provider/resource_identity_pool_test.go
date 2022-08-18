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
	scenarioStateIdentityPoolHasBeenCreated             = "The new identity_pool has been just created"
	scenarioStateIdentityPoolDescriptionHaveBeenUpdated = "The new identity_pool's description have been just updated"
	scenarioStateIdentityPoolHasBeenDeleted             = "The new identity_pool has been deleted"
	identityPoolScenarioName                            = "confluent_identity_pool Resource Lifecycle"
	identityPoolId                                      = "pool-AzXR"
	identityPoolDisplayName                             = "My Identity Pool"
	identityPoolDescription                             = "Prod Access to Kafka clusters to Release Engineering"
	identityPoolIdentityClaim                           = "claims.sub"
	identityPoolFilter                                  = "claims.aud==\"confluent\" && claims.group!=\"invalid_group\""
)

func TestAccIdentityPool(t *testing.T) {
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
	createSaResponse, _ := ioutil.ReadFile("../testdata/identity_pool/create_identity_pool.json")
	createSaStub := wiremock.Post(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools", identityProviderId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateIdentityPoolHasBeenCreated).
		WillReturn(
			string(createSaResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createSaStub)

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_created_identity_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenCreated).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedIdentityPoolResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_updated_identity_pool.json")
	patchSaStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenCreated).
		WillSetStateTo(scenarioStateIdentityPoolDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedIdentityPoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchSaStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedIdentityPoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedSaResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_deleted_identity_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolHasBeenDeleted).
		WillReturn(
			string(readDeletedSaResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteSaStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(identityPoolScenarioName).
		WhenScenarioStateIs(scenarioStateIdentityPoolDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateIdentityPoolHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteSaStub)

	// in order to test tf update (step #3)
	identityPoolUpdatedDescription := "Prod Access to Kafka clusters to Release Engineering updated"
	identityPoolUpdatedDisplayName := "My Identity Pool updated"
	identityPoolUpdatedIdentityClaim := "aud"
	identityPoolUpdatedFilter := "claims.aud==\"confluent\""
	identityPoolResourceLabel := "test_identity_pool_resource_label"
	fullIdentityPoolResourceLabel := fmt.Sprintf("confluent_identity_pool.%s", identityPoolResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckIdentityPoolDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckIdentityPoolConfig(mockServerUrl, identityPoolResourceLabel, identityPoolDisplayName, identityPoolDescription, identityPoolIdentityClaim, identityPoolFilter),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramId, identityPoolId),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramDisplayName, identityPoolDisplayName),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramDescription, identityPoolDescription),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramIdentityClaim, identityPoolIdentityClaim),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramFilter, identityPoolFilter),
				),
			},
			{
				Config: testAccCheckIdentityPoolConfig(mockServerUrl, identityPoolResourceLabel, identityPoolUpdatedDisplayName, identityPoolUpdatedDescription, identityPoolUpdatedIdentityClaim, identityPoolUpdatedFilter),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolResourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramId, identityPoolId),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramDisplayName, identityPoolUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramDescription, identityPoolUpdatedDescription),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramIdentityClaim, identityPoolUpdatedIdentityClaim),
					resource.TestCheckResourceAttr(fullIdentityPoolResourceLabel, paramFilter, identityPoolUpdatedFilter),
				),
			},
			{
				ResourceName:      fullIdentityPoolResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					poolId := resources[fullIdentityPoolResourceLabel].Primary.ID
					providerId := resources[fullIdentityPoolResourceLabel].Primary.Attributes["identity_provider.0.id"]
					return providerId + "/" + poolId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createSaStub, "POST /iam/v2/identity-providers/op-537/identity-pools", expectedCountOne)
	checkStubCount(t, wiremockClient, patchSaStub, "PATCH /iam/v2/identity-providers/op-537/identity-pools/pool-rORN", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteSaStub, "DELETE /iam/v2/identity-providers/op-537/identity-pools/pool-rORN", expectedCountOne)
}

func testAccCheckIdentityPoolDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each service account is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_identity_pool" {
			continue
		}
		deletedIdentityPoolId := rs.Primary.ID
		req := c.oidcClient.IdentityPoolsIamV2Api.GetIamV2IdentityPool(c.oidcApiContext(context.Background()), identityProviderId, deletedIdentityPoolId)
		deletedIdentityPool, response, err := req.Execute()
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(response)
		if isResourceNotFound {
			return nil
		} else if err == nil && deletedIdentityPool.Id != nil {
			// Otherwise return the error
			if *deletedIdentityPool.Id == rs.Primary.ID {
				return fmt.Errorf("service account (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckIdentityPoolConfig(mockServerUrl, identityPoolResourceLabel, identityPoolDisplayName, identityPoolDescription, identityPoolPrincipalClaim, identityPoolFilter string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_identity_pool" "%s" {
        identity_provider {
            id = "%s"
        }
		display_name    = "%s"
		description     = "%s"
		identity_claim  = "%s"
		filter          = %q
	}
	`, mockServerUrl, identityPoolResourceLabel, identityProviderId, identityPoolDisplayName, identityPoolDescription, identityPoolPrincipalClaim, identityPoolFilter)
}

func testAccCheckIdentityPoolExists(n string) resource.TestCheckFunc {
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
