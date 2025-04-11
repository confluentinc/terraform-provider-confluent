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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateComputePoolIsProvisioning = "The new compute pool is in provisioning state"
	scenarioStateComputePoolHasBeenCreated = "The new compute pool has been just created"
	scenarioStateComputePoolHasBeenDeleted = "The new compute pool has been deleted"
	flinkComputePoolScenarioName           = "confluent_flink_compute_pool Resource Lifecycle"
	flinkComputePoolCloud                  = "AWS"
	flinkComputePoolRegion                 = "us-east-2"
	flinkComputePoolEnvironmentId          = "env-gz903"
	flinkComputePoolResourceName           = "crn://confluent.cloud/organization=foo/environment=env-gz903/flink-region=aws.us-east-2/compute-pool=lfcp-abc123"
	flinkComputePoolId                     = "lfcp-abc123"
	flinkComputePoolDisplayName            = "flink_compute_pool_0"
	flinkComputePoolDefaultMaxCfu          = 5
	flinkComputePoolApiVersion             = "fcpm/v2"
	flinkComputePoolKind                   = "ComputePool"
	flinkComputePoolRestEndpoint           = "https://flink.us-east-2.aws.confluent.cloud/sql/v1alpha1/environments/env-gz903"
)

var flinkComputePoolUrlPath = fmt.Sprintf("/fcpm/v2/compute-pools/%s", flinkComputePoolId)

func TestAccComputePool(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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
	createComputePoolResponse, _ := ioutil.ReadFile("../testdata/compute_pool/create_compute_pool.json")
	createComputePoolStub := wiremock.Post(wiremock.URLPathEqualTo("/fcpm/v2/compute-pools")).
		InScenario(flinkComputePoolScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateComputePoolIsProvisioning).
		WillReturn(
			string(createComputePoolResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createComputePoolStub)

	readProvisioningComputePoolResponse, _ := ioutil.ReadFile("../testdata/compute_pool/read_provisioning_compute_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkComputePoolUrlPath)).
		InScenario(flinkComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(scenarioStateComputePoolIsProvisioning).
		WillSetStateTo(scenarioStateComputePoolHasBeenCreated).
		WillReturn(
			string(readProvisioningComputePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedComputePoolResponse, _ := ioutil.ReadFile("../testdata/compute_pool/read_created_compute_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkComputePoolUrlPath)).
		InScenario(flinkComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(scenarioStateComputePoolHasBeenCreated).
		WillReturn(
			string(readCreatedComputePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteComputePoolStub := wiremock.Delete(wiremock.URLPathEqualTo(flinkComputePoolUrlPath)).
		InScenario(flinkComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(scenarioStateComputePoolHasBeenCreated).
		WillSetStateTo(scenarioStateComputePoolHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteComputePoolStub)

	readDeletedComputePoolResponse, _ := ioutil.ReadFile("../testdata/compute_pool/read_deleted_compute_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkComputePoolUrlPath)).
		InScenario(flinkComputePoolScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkComputePoolEnvironmentId)).
		WhenScenarioStateIs(scenarioStateComputePoolHasBeenDeleted).
		WillReturn(
			string(readDeletedComputePoolResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	flinkComputePoolResourceLabel := "test"
	fullComputePoolResourceLabel := fmt.Sprintf("confluent_flink_compute_pool.%s", flinkComputePoolResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckComputePoolDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckComputePoolConfig(mockServerUrl, flinkComputePoolResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolExists(fullComputePoolResourceLabel),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramId, flinkComputePoolId),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramDisplayName, flinkComputePoolDisplayName),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramCloud, flinkComputePoolCloud),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramRegion, flinkComputePoolRegion),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramMaxCfu, strconv.Itoa(flinkComputePoolDefaultMaxCfu)),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkComputePoolEnvironmentId),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramApiVersion, flinkComputePoolApiVersion),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramKind, flinkComputePoolKind),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramResourceName, flinkComputePoolResourceName),
				),
			},
			{
				Config: testAccCheckComputePoolConfigWithoutMaxCfu(mockServerUrl, flinkComputePoolResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolExists(fullComputePoolResourceLabel),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramId, flinkComputePoolId),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramDisplayName, flinkComputePoolDisplayName),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramCloud, flinkComputePoolCloud),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramRegion, flinkComputePoolRegion),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramMaxCfu, strconv.Itoa(flinkComputePoolDefaultMaxCfu)),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkComputePoolEnvironmentId),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramApiVersion, flinkComputePoolApiVersion),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramKind, flinkComputePoolKind),
					resource.TestCheckResourceAttr(fullComputePoolResourceLabel, paramResourceName, flinkComputePoolResourceName),
				),
			},
			{
				ResourceName:      fullComputePoolResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					flinkComputePoolId := resources[fullComputePoolResourceLabel].Primary.ID
					environmentId := resources[fullComputePoolResourceLabel].Primary.Attributes["environment.0.id"]
					return environmentId + "/" + flinkComputePoolId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createComputePoolStub, fmt.Sprintf("POST %s", flinkComputePoolUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteComputePoolStub, fmt.Sprintf("DELETE %s?environment=%s", flinkComputePoolUrlPath, flinkComputePoolEnvironmentId), expectedCountOne)
}

func testAccCheckComputePoolDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each compute pool is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_compute_pool" {
			continue
		}
		deletedComputePoolId := rs.Primary.ID
		req := c.netClient.NetworksNetworkingV1Api.GetNetworkingV1Network(c.netApiContext(context.Background()), deletedComputePoolId).Environment(flinkComputePoolEnvironmentId)
		deletedComputePool, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedComputePool.Id != nil {
			// Otherwise return the error
			if *deletedComputePool.Id == rs.Primary.ID {
				return fmt.Errorf("compute pool (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckComputePoolConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_flink_compute_pool" "%s" {
        display_name     = "%s"
        cloud            = "%s"
	    region           = "%s"
	    environment {
		  id = "%s"
	    }
        max_cfu = %d
	}
	`, mockServerUrl, resourceLabel, flinkComputePoolDisplayName, flinkComputePoolCloud, flinkComputePoolRegion, flinkComputePoolEnvironmentId, flinkComputePoolDefaultMaxCfu)
}

func testAccCheckComputePoolConfigWithoutMaxCfu(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_flink_compute_pool" "%s" {
	    display_name            = "%s"
	    cloud            = "%s"
	    region           = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, flinkComputePoolDisplayName, flinkComputePoolCloud, flinkComputePoolRegion, flinkComputePoolEnvironmentId)
}

func testAccCheckComputePoolExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s compute pool has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s compute pool", n)
		}

		return nil
	}
}
