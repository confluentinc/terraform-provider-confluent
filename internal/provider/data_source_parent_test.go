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
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	parentDataSourceScenarioName = "confluent_parent Data Source Lifecycle"
	parentDataSourceLabel        = "test_parent_data_source_label"

	expectedParentResourceName = "crn://confluent.cloud/parent=1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	expectedParentId           = "1111aaaa-11aa-11aa-11aa-111111aaaaaa"
)

var fullParentDataSourceLabel = fmt.Sprintf("data.confluent_parent.%s", parentDataSourceLabel)

func TestAccDataSourceParent(t *testing.T) {
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

	readParentResponse, _ := ioutil.ReadFile("../testdata/parent/read_parent.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/parent/1111aaaa-11aa-11aa-11aa-111111aaaaaa")).
		InScenario(parentDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readParentResponse),
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
				Config: testAccCheckDataSourceParentConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckParentExists(fullParentDataSourceLabel),
					resource.TestCheckResourceAttr(fullParentDataSourceLabel, "id", expectedParentId),
					resource.TestCheckResourceAttr(fullParentDataSourceLabel, "resource_name", expectedParentResourceName),
					resource.TestCheckResourceAttr(fullParentDataSourceLabel, "invitation_restrictions_enabled", "true"),
				),
			},
		},
	})
}

func testAccCheckDataSourceParentConfig(confluentCloudBaseUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
    }
	data "confluent_parent" "%s" {
		id = "1111aaaa-11aa-11aa-11aa-111111aaaaaa"
	}
	`, confluentCloudBaseUrl, parentDataSourceLabel)
}

func testAccCheckParentExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s parent has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s parent", n)
		}

		return nil
	}
}
