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
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	invitationResourceScenarioName        = "confluent_invitation Data Source Lifecycle"
	scenarioStateInvitationHasBeenCreated = "A new invitation has been just created"
	invitationEmail                       = "zli00000000@confluent.io"
	invitationResourceLabel               = "confluent_invitation.inv"

	createInvitationUrlPath      = "/iam/v2/invitations"
	readCreatedInvitationUrlPath = "/iam/v2/invitations/i-7od91"
)

func TestAccInvitation(t *testing.T) {
	mockServerUrl := tc.wiremockUrl
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createInvitationResponse, _ := ioutil.ReadFile("../testdata/invitation/create_invitation.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createInvitationUrlPath)).
		InScenario(invitationResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateInvitationHasBeenCreated).
		WillReturn(
			string(createInvitationResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedInvitationUrlPath)).
		InScenario(invitationResourceScenarioName).
		WhenScenarioStateIs(scenarioStateInvitationHasBeenCreated).
		WillReturn(
			string(createInvitationResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readCreatedInvitationUrlPath)).
		InScenario(invitationResourceScenarioName).
		WhenScenarioStateIs(scenarioStateInvitationHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceInvitationWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(invitationResourceLabel, "email", "zli00000000@confluent.io"),
					resource.TestCheckResourceAttr(invitationResourceLabel, "id", "i-7od91"),
					resource.TestCheckResourceAttr(invitationResourceLabel, "creator.0.id", "u-5m00y8"),
					resource.TestCheckResourceAttr(invitationResourceLabel, "user.0.id", "u-r0jy59"),
					resource.TestCheckResourceAttr(invitationResourceLabel, "auth_type", "AUTH_TYPE_LOCAL"),
					resource.TestCheckResourceAttr(invitationResourceLabel, "status", "INVITE_STATUS_SENT"),
				),
			},
		},
	})
}

func testAccCheckResourceInvitationWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_invitation" "inv" {
		email = "%s"
	}
	`, mockServerUrl, invitationEmail)
}
