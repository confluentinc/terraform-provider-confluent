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
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateEnvHasBeenCreated     = "The new environment has been just created"
	scenarioStateEnvNameHasBeenUpdated = "The new environment's name has been just updated"
	scenarioStateEnvHasBeenDeleted     = "The new environment has been deleted"
	envScenarioName                    = "confluent_environment Resource Lifecycle"
	expectedCountZero                  = int64(0)
	expectedCountOne                   = int64(1)
	expectedCountTwo                   = int64(2)
)

var contentTypeJSONHeader = map[string]string{"Content-Type": "application/json"}

func TestAccEnvironment(t *testing.T) {
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
	createEnvResponse, _ := ioutil.ReadFile("../testdata/environment/create_env.json")
	createEnvStub := wiremock.Post(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateEnvHasBeenCreated).
		WillReturn(
			string(createEnvResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createEnvStub)

	readCreatedEnvResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments/env-q2opmd")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(scenarioStateEnvHasBeenCreated).
		WillReturn(
			string(readCreatedEnvResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedEnvResponse, _ := ioutil.ReadFile("../testdata/environment/read_updated_env.json")
	patchEnvStub := wiremock.Patch(wiremock.URLPathEqualTo("/org/v2/environments/env-q2opmd")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(scenarioStateEnvHasBeenCreated).
		WillSetStateTo(scenarioStateEnvNameHasBeenUpdated).
		WillReturn(
			string(readUpdatedEnvResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchEnvStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments/env-q2opmd")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(scenarioStateEnvNameHasBeenUpdated).
		WillReturn(
			string(readUpdatedEnvResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedEnvResponse, _ := ioutil.ReadFile("../testdata/environment/read_deleted_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments/env-q2opmd")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(scenarioStateEnvHasBeenDeleted).
		WillReturn(
			string(readDeletedEnvResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	deleteEnvStub := wiremock.Delete(wiremock.URLPathEqualTo("/org/v2/environments/env-q2opmd")).
		InScenario(envScenarioName).
		WhenScenarioStateIs(scenarioStateEnvNameHasBeenUpdated).
		WillSetStateTo(scenarioStateEnvHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteEnvStub)

	environmentDisplayName := "test_env_display_name"
	// in order to test tf update (step #3)
	environmentDisplayUpdatedName := "test_env_display_updated_name"
	environmentResourceLabel := "test_env_resource_label"
	environmentResourceEndpoint := "crn://confluent.cloud/organization=foo/environment=env-q2opmd"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckEnvironmentDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckEnvironmentConfig(mockServerUrl, environmentResourceLabel, environmentDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "id", "env-q2opmd"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "display_name", environmentDisplayName),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fmt.Sprintf("confluent_environment.%s", environmentResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckEnvironmentConfig(mockServerUrl, environmentResourceLabel, environmentDisplayUpdatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "id", "env-q2opmd"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "display_name", environmentDisplayUpdatedName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_environment.%s", environmentResourceLabel), "resource_name", environmentResourceEndpoint),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_environment.%s", environmentResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createEnvStub, "POST /org/v2/environments", expectedCountOne)
	checkStubCount(t, wiremockClient, patchEnvStub, "PATCH /org/v2/environments/env-q2opmd", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteEnvStub, "DELETE /org/v2/environments/env-q2opmd", expectedCountOne)
}

func testAccCheckEnvironmentDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each environment is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_environment" {
			continue
		}
		deletedEnvironmentId := rs.Primary.ID
		req := c.orgClient.EnvironmentsOrgV2Api.GetOrgV2Environment(c.orgApiContext(context.Background()), deletedEnvironmentId)
		deletedEnvironment, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/environments/{nonExistentEnvId/deletedEnvID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the environment is destroyed.
			return nil
		} else if err == nil && deletedEnvironment.Id != nil {
			// Otherwise return the error
			if *deletedEnvironment.Id == rs.Primary.ID {
				return fmt.Errorf("environment (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckEnvironmentConfig(mockServerUrl, environmentResourceLabel, environmentDisplayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_environment" "%s" {
		display_name = "%s"
	}
	`, mockServerUrl, environmentResourceLabel, environmentDisplayName)
}

func testAccCheckEnvironmentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s environment has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s environment", n)
		}

		return nil
	}
}

func checkStubCount(t *testing.T, client *wiremock.Client, rule *wiremock.StubRule, requestTypeAndEndpoint string, expectedCount int64) {
	verifyStub, _ := client.Verify(rule.Request(), expectedCount)
	actualCount, _ := client.GetCountRequests(rule.Request())
	if !verifyStub {
		t.Fatalf("expected %#v %s requests but found %#v", expectedCount, requestTypeAndEndpoint, actualCount)
	}
}
