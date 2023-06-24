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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	invitationDataSourceScenarioName = "confluent_invitation Data Source Lifecycle"

	invitationUrlPath = "/iam/v2/invitations/i-gxxn1"
	invitationId      = "i-gxxn1"
	invitationLabel   = "data.confluent_invitation.inv"
)

func TestAccDataSourceInvitation(t *testing.T) {
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

	readInvitationResponse, _ := ioutil.ReadFile("../testdata/invitation/read_invitation.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(invitationUrlPath)).
		InScenario(invitationDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readInvitationResponse),
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
				Config: testAccCheckDataSourceInvitationWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(invitationLabel, "email", "zli@confluent.io"),
					resource.TestCheckResourceAttr(invitationLabel, "id", "i-gxxn1"),
					resource.TestCheckResourceAttr(invitationLabel, "creator.0.id", "u-5m00y8"),
					resource.TestCheckResourceAttr(invitationLabel, "user.0.id", "u-ox23ro"),
					resource.TestCheckResourceAttr(invitationLabel, "auth_type", "AUTH_TYPE_LOCAL"),
					resource.TestCheckResourceAttr(invitationLabel, "status", "INVITE_STATUS_SENT"),
				),
			},
		},
	})
}

func testAccCheckDataSourceInvitationWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_invitation" "inv" {
		id = "%s"
	}
	`, mockServerUrl, invitationId)
}
