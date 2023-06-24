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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	usersDataSourceScenarioName = "confluent_users Data Source Lifecycle"
)

var userIds = []string{"u-1jjv21", "u-1jjv22", "u-1jjv23"}

func TestAccDataSourceUsers(t *testing.T) {
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

	readUsersPageOneResponse, _ := ioutil.ReadFile("../testdata/user/read_users_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listUsersPageSize))).
		InScenario(usersDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readUsersPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUsersPageTwoResponse, _ := ioutil.ReadFile("../testdata/user/read_users_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listUsersPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(userLastPagePageToken)).
		InScenario(usersDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readUsersPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullUserDataSourceLabel := fmt.Sprintf("data.confluent_users.%s", userResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceUsers(mockServerUrl, userResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUsersExists(fullUserDataSourceLabel),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, fmt.Sprintf("%s.#", paramIds), "3"),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, fmt.Sprintf("%s.0", paramIds), userIds[0]),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, fmt.Sprintf("%s.1", paramIds), userIds[1]),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, fmt.Sprintf("%s.2", paramIds), userIds[2]),
				),
			},
		},
	})
}

func testAccCheckDataSourceUsers(mockServerUrl, userResourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_users" "%s" {
	}
	`, mockServerUrl, userResourceLabel)
}

func testAccCheckUsersExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s user has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s user", n)
		}

		return nil
	}
}
