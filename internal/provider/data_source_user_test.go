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
	userApiVersion             = "iam/v2"
	userDataSourceScenarioName = "confluent_user Data Source Lifecycle"
	userId                     = "u-1jjv23"
	userEmail                  = "test3@gmail.com"
	userFullName               = "Alex #3"
	userResourceLabel          = "test_user_resource_label"
	userLastPagePageToken      = "dyJpZCI6InNhLTd5OXbyby"
)

func TestAccDataSourceUser(t *testing.T) {
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

	readCreatedUserResponse, _ := ioutil.ReadFile("../testdata/user/read_created_user.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users/u-1jjv23")).
		InScenario(userDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedUserResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUsersPageOneResponse, _ := ioutil.ReadFile("../testdata/user/read_users_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listUsersPageSize))).
		InScenario(userDataSourceScenarioName).
		WillReturn(
			string(readUsersPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUsersPageTwoResponse, _ := ioutil.ReadFile("../testdata/user/read_users_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listUsersPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(userLastPagePageToken)).
		InScenario(userDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readUsersPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullUserDataSourceLabel := fmt.Sprintf("data.confluent_user.%s", userResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceUserConfigWithIdSet(mockServerUrl, userResourceLabel, userId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(fullUserDataSourceLabel),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramId, userId),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramApiVersion, userApiVersion),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramKind, userKind),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramEmail, userEmail),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramFullName, userFullName),
				),
			},
			{
				Config: testAccCheckDataSourceUserConfigWithEmailSet(mockServerUrl, userResourceLabel, userEmail),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(fullUserDataSourceLabel),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramId, userId),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramApiVersion, userApiVersion),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramKind, userKind),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramEmail, userEmail),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramFullName, userFullName),
				),
			},
			{
				Config: testAccCheckDataSourceUserConfigWithFullNameSet(mockServerUrl, userResourceLabel, userFullName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(fullUserDataSourceLabel),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramId, userId),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramApiVersion, userApiVersion),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramKind, userKind),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramEmail, userEmail),
					resource.TestCheckResourceAttr(fullUserDataSourceLabel, paramFullName, userFullName),
				),
			},
		},
	})
}

func testAccCheckDataSourceUserConfigWithIdSet(mockServerUrl, userResourceLabel, userId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_user" "%s" {
		id = "%s"
	}
	`, mockServerUrl, userResourceLabel, userId)
}

func testAccCheckDataSourceUserConfigWithEmailSet(mockServerUrl, userResourceLabel, email string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_user" "%s" {
		email = "%s"
	}
	`, mockServerUrl, userResourceLabel, email)
}

func testAccCheckDataSourceUserConfigWithFullNameSet(mockServerUrl, userResourceLabel, fullName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_user" "%s" {
		full_name = "%s"
	}
	`, mockServerUrl, userResourceLabel, fullName)
}

func testAccCheckUserExists(n string) resource.TestCheckFunc {
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
