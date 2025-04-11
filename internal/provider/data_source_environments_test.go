// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	environmentsDataSourceScenarioName = "confluent_environments Data Source Lifecycle"
	envResourceLabel                   = "test_env_resource_label"
)

var environmentIds = []string{"env-1jnw8z", "env-7n1r31", "env-prp21o"}

func TestAccDataSourceEnvironments(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	readEnvironmentsPageOneResponse, _ := ioutil.ReadFile("../testdata/environment/read_envs_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/org/v2/environments")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listEnvironmentsPageSize))).
		InScenario(environmentsDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readEnvironmentsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullEnvironmentDataSourceLabel := fmt.Sprintf("data.confluent_environments.%s", envResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceEnvironments(mockServerUrl, envResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckEnvironmentsExists(fullEnvironmentDataSourceLabel),
					resource.TestCheckResourceAttr(fullEnvironmentDataSourceLabel, fmt.Sprintf("%s.#", paramIds), "3"),
					resource.TestCheckResourceAttr(fullEnvironmentDataSourceLabel, fmt.Sprintf("%s.0", paramIds), environmentIds[0]),
					resource.TestCheckResourceAttr(fullEnvironmentDataSourceLabel, fmt.Sprintf("%s.1", paramIds), environmentIds[1]),
					resource.TestCheckResourceAttr(fullEnvironmentDataSourceLabel, fmt.Sprintf("%s.2", paramIds), environmentIds[2]),
				),
			},
		},
	})
	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})
}

func testAccCheckDataSourceEnvironments(mockServerUrl, envResourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_environments" "%s" {
	}
	`, mockServerUrl, envResourceLabel)
}

func testAccCheckEnvironmentsExists(n string) resource.TestCheckFunc {
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
