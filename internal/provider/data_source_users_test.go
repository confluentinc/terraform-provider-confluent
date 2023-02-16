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
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	usersAPIVersion             = "iam/v2"
	usersDataSourceScenarioName = "confluent_user Data Source Lifecycle"
	usersID                     = "u-1jjv21"
	usersEmail                  = "test1@gmail.com"
	usersFullName               = "Alex #1"
	usersResourceLabel          = "test_users_resource_label"
	usersLastPagePageToken      = "dyJpZCI6InNhLTd5OXbyby"
)

func TestAccDataSourceUsers(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockServerURL := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerURL)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readUsersPageOneResponse, _ := ioutil.ReadFile("../testdata/user/read_users_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/users")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listUsersPageSize))).
		InScenario(usersDataSourceScenarioName).
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

	fullUsersDataSourceLabel := fmt.Sprintf("data.confluent_users.%s", usersResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceUsersConfig(mockServerURL, usersResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullUsersDataSourceLabel, "id"),
					resource.TestCheckTypeSetElemNestedAttrs(fullUsersDataSourceLabel, "users.*", map[string]string{
						paramId:         usersID,
						paramApiVersion: usersAPIVersion,
						paramKind:       userKind,
						paramEmail:      usersEmail,
						paramFullName:   usersFullName,
					}),
					resource.TestCheckResourceAttr(fullUsersDataSourceLabel, "users.0.id", usersID),
					resource.TestCheckResourceAttr(fullUsersDataSourceLabel, "users.0.api_version", usersAPIVersion),
					resource.TestCheckResourceAttr(fullUsersDataSourceLabel, "users.0.kind", userKind),
					resource.TestCheckResourceAttr(fullUsersDataSourceLabel, "users.0.email", usersEmail),
					resource.TestCheckResourceAttr(fullUsersDataSourceLabel, "users.0.full_name", usersFullName),
				),
			},
		},
	})
}

func testAccCheckDataSourceUsersConfig(mockServerURL, usersResourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_users" "%s" {}
	`, mockServerURL, usersResourceLabel)
}
