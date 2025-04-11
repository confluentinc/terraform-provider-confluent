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
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	groupMappingDataSourceScenarioName = "confluent_group_mapping Data Source Lifecycle"
	groupMappingLastPagePageToken      = "eyJjcmVhdGVkVGltZSI6WzIwM"
)

func TestAccDataSourceGroupMapping(t *testing.T) {
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

	readCreatedSaResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_created_group_mapping.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings/group-w4vP")).
		InScenario(groupMappingDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedSaResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGroupMappingsPageOneResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_group_mappings_page_1.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listGroupMappingsPageSize))).
		InScenario(groupMappingDataSourceScenarioName).
		WillReturn(
			string(readGroupMappingsPageOneResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readGroupMappingsPageTwoResponse, _ := ioutil.ReadFile("../testdata/group_mapping/read_group_mappings_page_2.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/sso/group-mappings")).
		WithQueryParam("page_size", wiremock.EqualTo(strconv.Itoa(listGroupMappingsPageSize))).
		WithQueryParam("page_token", wiremock.EqualTo(groupMappingLastPagePageToken)).
		InScenario(groupMappingDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readGroupMappingsPageTwoResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	fullGroupMappingDataSourceLabel := fmt.Sprintf("data.confluent_group_mapping.%s", groupMappingResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGroupMappingConfigWithIdSet(mockServerUrl, groupMappingResourceLabel, groupMappingId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingExists(fullGroupMappingDataSourceLabel),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramId, groupMappingId),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramDisplayName, groupMappingDisplayName),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramFilter, groupMappingFilter),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramDescription, groupMappingDescription),
				),
			},
			{
				Config: testAccCheckDataSourceGroupMappingConfigWithDisplayNameSet(mockServerUrl, groupMappingResourceLabel, groupMappingDisplayName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGroupMappingExists(fullGroupMappingDataSourceLabel),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramId, groupMappingId),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramDisplayName, groupMappingDisplayName),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramFilter, groupMappingFilter),
					resource.TestCheckResourceAttr(fullGroupMappingDataSourceLabel, paramDescription, groupMappingDescription),
				),
			},
		},
	})
}

func testAccCheckDataSourceGroupMappingConfigWithIdSet(mockServerUrl, gmResourceLabel, gmId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_group_mapping" "%s" {
		id = "%s"
	}
	`, mockServerUrl, gmResourceLabel, gmId)
}

func testAccCheckDataSourceGroupMappingConfigWithDisplayNameSet(mockServerUrl, gmResourceLabel, displayName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	data "confluent_group_mapping" "%s" {
		display_name = "%s"
	}
	`, mockServerUrl, gmResourceLabel, displayName)
}
