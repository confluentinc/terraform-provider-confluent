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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

var fullIdentityPoolDataSourceLabel = fmt.Sprintf("data.confluent_identity_pool.%s", identityPoolDataSourceLabel)

func TestAccDataSourceIdentityPool(t *testing.T) {
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

	readCreatedIdentityPoolResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_created_identity_pool.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools/%s", identityProviderId, identityPoolId))).
		InScenario(dataSourceIdentityPoolScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedIdentityPoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	readIdentityPoolsResponse, _ := ioutil.ReadFile("../testdata/identity_pool/read_identity_pools.json")
	if err := wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/iam/v2/identity-providers/%s/identity-pools", identityProviderId))).
		InScenario(dataSourceIdentityPoolScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readIdentityPoolsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)); err != nil {
		t.Logf("StubFor failed: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAwsIdentityPoolConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolDataSourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramId, identityPoolId),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramDisplayName, identityPoolDisplayName),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramDescription, identityPoolDescription),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramIdentityClaim, identityPoolIdentityClaim),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramFilter, identityPoolFilter),
				),
			},
			{
				Config: testAccCheckDataSourceAzureIdentityPoolConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckIdentityPoolExists(fullIdentityPoolDataSourceLabel),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramId, identityPoolId),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramDisplayName, identityPoolDisplayName),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramDescription, identityPoolDescription),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramIdentityClaim, identityPoolIdentityClaim),
					resource.TestCheckResourceAttr(fullIdentityPoolDataSourceLabel, paramFilter, identityPoolFilter),
				),
			},
		},
	})
}

func testAccCheckDataSourceAzureIdentityPoolConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_identity_pool" "%s" {
		display_name = "%s"
	  	identity_provider {
			id = "%s"
	  	}
	}
	`, mockServerUrl, identityPoolDataSourceLabel, identityPoolDisplayName, identityProviderId)
}

func testAccCheckDataSourceAwsIdentityPoolConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_identity_pool" "%s" {
	    id = "%s"
	    identity_provider {
		  id = "%s"
	    }
	}
	`, mockServerUrl, networkDataSourceLabel, identityPoolId, identityProviderId)
}
