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
)

const (
	envScenarioDataSourceName        = "confluent_environment Data Source Lifecycle"
	environmentDataSourceDisplayName = "test_env_display_name"
	environmentDataSourceLabel       = "test_env_data_source_label"
	environmentDataSourceEndpoint    = "crn://confluent.cloud/organization=foo/environment=env-1jrymj"
)

func TestAccDataSourceEnvironment(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()
	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readCreatedEnvResponse, _ := ioutil.ReadFile("../testdata/environment/read_created_env.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/org/v2/environments/%s", testEnvironmentId))).
		InScenario(envScenarioDataSourceName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedEnvResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readEnvironmentsResponse, _ := ioutil.ReadFile("../testdata/environment/read_envs.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		InScenario(envScenarioDataSourceName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readEnvironmentsResponse),
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
				Config: testAccCheckDataSourceEnvironmentConfigWithIdSet(mockServerUrl, environmentDataSourceLabel, testEnvironmentId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), paramId, testEnvironmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), paramDisplayName, environmentDataSourceDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), getNestedStreamGovernancePackageKey(), "ESSENTIALS"),
				),
			},
			{
				Config: testAccCheckDataSourceEnvironmentConfigWithDisplayNameSet(mockServerUrl, environmentDataSourceLabel, environmentDataSourceDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentExists(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), paramId, testEnvironmentId),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), paramDisplayName, environmentDataSourceDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), paramResourceName, environmentDataSourceEndpoint),
					resource.TestCheckResourceAttr(fmt.Sprintf("data.confluent_environment.%s", environmentDataSourceLabel), getNestedStreamGovernancePackageKey(), ""),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourceEnvironmentConfigWithIdSet(mockServerUrl, environmentDataSourceLabel, testEnvironmentId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_environment" "%s" {
		id = "%s"
	}
	`, mockServerUrl, environmentDataSourceLabel, testEnvironmentId)
}

func testAccCheckDataSourceEnvironmentConfigWithDisplayNameSet(mockServerUrl, environmentDataSourceLabel, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_environment" "%s" {
		display_name = "%s"
	}
	`, mockServerUrl, environmentDataSourceLabel, displayName)
}
