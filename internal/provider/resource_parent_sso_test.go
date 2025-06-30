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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateParentSSOHasBeenCreated = "The new parent SSO has been just created"
	scenarioStateParentSSOHasBeenUpdated = "The parent SSO has been updated"
	scenarioStateParentSSOHasBeenDeleted = "The requested parent SSO has been deleted"
	parentSSOScenarioName                = "confluent_parent_sso Resource Lifecycle"
	parentSSOId                          = "5d457a1e-6314-4867-95bc-e84ef2d6eaaa"
	parentSSOUrlPath                     = "/iam/v2/parent-sso/5d457a1e-6314-4867-95bc-e84ef2d6eaaa"

	// Initial values
	pssoConnectionName        = "confluent-sso"
	pssoEmailAttributeMapping = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
	pssoSignInEndpoint        = "https://okta.com/app/abcdef/sso/saml"
	pssoSigningCert           = "cert-abc"
	pssoResourceLabel         = "test_parent_sso_resource_label"

	// Updated values
	pssoUpdatedEmailAttributeMapping = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddressupdated"
	pssoUpdatedSignInEndpoint        = "https://okta.com/app/updated/sso/saml"
	pssoUpdatedSigningCert           = "cert-abc-updated"
)

func TestAccParentSSO(t *testing.T) {
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

	createParentSSOResponse, _ := ioutil.ReadFile("../testdata/parent_sso/create_parent_sso.json")
	createParentSSOStub := wiremock.Post(wiremock.URLPathEqualTo("/iam/v2/parent-sso")).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateParentSSOHasBeenCreated).
		WillReturn(
			string(createParentSSOResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createParentSSOStub)

	readCreatedParentSSOResponse, _ := ioutil.ReadFile("../testdata/parent_sso/read_created_parent_sso.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(parentSSOUrlPath)).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(scenarioStateParentSSOHasBeenCreated).
		WillReturn(
			string(readCreatedParentSSOResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateParentSSOResponse, _ := ioutil.ReadFile("../testdata/parent_sso/read_updated_parent_sso.json")
	updateParentSSOStub := wiremock.Patch(wiremock.URLPathEqualTo(parentSSOUrlPath)).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(scenarioStateParentSSOHasBeenCreated).
		WillSetStateTo(scenarioStateParentSSOHasBeenUpdated).
		WillReturn(
			string(updateParentSSOResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateParentSSOStub)

	readUpdatedParentSSOResponse, _ := ioutil.ReadFile("../testdata/parent_sso/read_updated_parent_sso.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(parentSSOUrlPath)).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(scenarioStateParentSSOHasBeenUpdated).
		WillReturn(
			string(readUpdatedParentSSOResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedParentSSOResponse, _ := ioutil.ReadFile("../testdata/parent_sso/read_deleted_parent_sso.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(parentSSOUrlPath)).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(scenarioStateParentSSOHasBeenDeleted).
		WillReturn(
			string(readDeletedParentSSOResponse),
			contentTypeJSONHeader,
			http.StatusForbidden,
		))

	deleteParentSSOStub := wiremock.Delete(wiremock.URLPathEqualTo(parentSSOUrlPath)).
		InScenario(parentSSOScenarioName).
		WhenScenarioStateIs(scenarioStateParentSSOHasBeenUpdated).
		WillSetStateTo(scenarioStateParentSSOHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteParentSSOStub)

	fullParentSSOResourceLabel := fmt.Sprintf("confluent_parent_sso.%s", pssoResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckParentSSODestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckParentSSOConfig(mockServerUrl, pssoResourceLabel, pssoConnectionName, pssoEmailAttributeMapping, "true", "true", "false", pssoSignInEndpoint, pssoSigningCert),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckParentSSOExists(fullParentSSOResourceLabel),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "id", parentSSOId),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "connection_name", pssoConnectionName),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "email_attribute_mapping", pssoEmailAttributeMapping),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "idp_initiated", "true"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "jit_enabled", "true"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "bup_enabled", "false"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "sign_in_endpoint", pssoSignInEndpoint),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:            fullParentSSOResourceLabel,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"signing_cert"}, // signing_cert is not returned by API
			},
			{
				// Test updating all supported attributes
				Config: testAccCheckParentSSOConfig(mockServerUrl, pssoResourceLabel, pssoConnectionName, pssoUpdatedEmailAttributeMapping, "false", "false", "true", pssoUpdatedSignInEndpoint, pssoUpdatedSigningCert),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckParentSSOExists(fullParentSSOResourceLabel),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "id", parentSSOId),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "connection_name", pssoConnectionName),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "email_attribute_mapping", pssoUpdatedEmailAttributeMapping),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "idp_initiated", "false"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "jit_enabled", "false"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "bup_enabled", "true"),
					resource.TestCheckResourceAttr(fullParentSSOResourceLabel, "sign_in_endpoint", pssoUpdatedSignInEndpoint),
				),
			},
		},
	})

	checkStubCount(t, wiremockClient, createParentSSOStub, "POST /iam/v2/parent-sso", expectedCountOne)
	checkStubCount(t, wiremockClient, updateParentSSOStub, fmt.Sprintf("PATCH /iam/v2/parent-sso/%s", parentSSOId), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteParentSSOStub, fmt.Sprintf("DELETE /iam/v2/parent-sso/%s", parentSSOId), expectedCountOne)
}

func testAccCheckParentSSODestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each parent SSO is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_parent_sso" {
			continue
		}
		deletedParentSSOId := rs.Primary.ID
		req := c.parentClient.ParentSSOsIamV2Api.GetIamV2ParentSSO(c.parentApiContext(context.Background()), deletedParentSSOId)
		deletedParentSSO, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedParentSSO.Id != nil {
			// Otherwise return the error
			if *deletedParentSSO.Id == rs.Primary.ID {
				return fmt.Errorf("parent SSO (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckParentSSOConfig(mockServerUrl, label, connectionName, emailAttributeMapping, idpInitiated, jitEnabled, bupEnabled, signInEndpoint, signingCert string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }
    resource "confluent_parent_sso" "%s" {
        connection_name = "%s"
        email_attribute_mapping = "%s"
        idp_initiated = %s
        jit_enabled = %s
        bup_enabled = %s
        sign_in_endpoint = "%s"
        signing_cert = "%s"
    }
    `, mockServerUrl, label, connectionName, emailAttributeMapping, idpInitiated, jitEnabled, bupEnabled, signInEndpoint, signingCert)
}

func testAccCheckParentSSOExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s parent SSO has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s parent SSO", n)
		}

		return nil
	}
}
