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
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	saApiVersion             = "iam/v2"
	saDataSourceScenarioName = "confluent_service_account Data Source Lifecycle"
	saId                     = "sa-1jjv26"
	saDisplayName            = "test_service_account_display_name"
	saDescription            = "The initial description of service account"
	saKind                   = "ServiceAccount"
	saResourceLabel          = "test_sa_resource_label"
	saLastPagePageToken      = "dyJpZCI6InNhLTd5OXbyby"
)

func TestAccDataSourceServiceAccount(t *testing.T) {
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

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/service_account/read_created_sa.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts/sa-1jjv26")).
		InScenario(saDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readServiceAccountsPageOneResponse, _ := ioutil.ReadFile("../testdata/service_account/read_sas_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listServiceAccountsPageSize))).
		InScenario(saDataSourceScenarioName).
		WillReturn(
			string(readServiceAccountsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readServiceAccountsPageTwoResponse, _ := ioutil.ReadFile("../testdata/service_account/read_sas_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/service-accounts")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listServiceAccountsPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(saLastPagePageToken)).
		InScenario(saDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readServiceAccountsPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullServiceAccountDataSourceLabel := fmt.Sprintf("data.confluent_service_account.%s", saResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceServiceAccountConfigWithIdSet(mockServerUrl, saResourceLabel, saId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullServiceAccountDataSourceLabel),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramId, saId),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramApiVersion, saApiVersion),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramKind, saKind),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramDisplayName, saDisplayName),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramDescription, saDescription),
				),
			},
			{
				Config: testAccCheckDataSourceServiceAccountConfigWithDisplayNameSet(mockServerUrl, saResourceLabel, saDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckServiceAccountExists(fullServiceAccountDataSourceLabel),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramId, saId),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramApiVersion, saApiVersion),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramKind, saKind),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramDisplayName, saDisplayName),
					resource.TestCheckResourceAttr(fullServiceAccountDataSourceLabel, paramDescription, saDescription),
				),
			},
		},
	})
}

func testAccCheckDataSourceServiceAccountConfigWithIdSet(mockServerUrl, saResourceLabel, saId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_service_account" "%s" {
		id = "%s"
	}
	`, mockServerUrl, saResourceLabel, saId)
}

func testAccCheckDataSourceServiceAccountConfigWithDisplayNameSet(mockServerUrl, saResourceLabel, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_service_account" "%s" {
		display_name = "%s"
	}
	`, mockServerUrl, saResourceLabel, displayName)
}
