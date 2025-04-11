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
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioStateCustomConnectorPluginPresignedUrlHasBeenCreated = "The new custom connector plugin's presigned URL has been just created"
	scenarioStateCustomConnectorPluginHasBeenCreated             = "The new custom connector plugin has been just created"
	scenarioStateCustomConnectorPluginDescriptionHaveBeenUpdated = "The new custom connector plugin's description and display name have been just updated"
	scenarioStateCustomConnectorPluginHasBeenDeleted             = "The new custom connector plugin has been deleted"
	customConnectorPluginScenarioName                            = "confluent_custom_connector_plugin Resource Lifecycle"
)

func TestAccCustomConnectorPlugin(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createCustomConnectorPluginPresignedUrlResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin/read_presigned_url.json")
	createCustomConnectorPluginPresignedUrlStub := wiremock.Post(wiremock.URLPathEqualTo("/connect/v1/presigned-upload-url")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCustomConnectorPluginPresignedUrlHasBeenCreated).
		WillReturn(
			string(createCustomConnectorPluginPresignedUrlResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCustomConnectorPluginPresignedUrlStub)

	createCustomConnectorPluginResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin/create_plugin.json")
	createCustomConnectorPluginStub := wiremock.Post(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginPresignedUrlHasBeenCreated).
		WillSetStateTo(scenarioStateCustomConnectorPluginHasBeenCreated).
		WillReturn(
			string(createCustomConnectorPluginResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCustomConnectorPluginStub)

	readCreatedCustomConnectorPluginResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin/read_created_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins/ccp-4rrw00")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginHasBeenCreated).
		WillReturn(
			string(readCreatedCustomConnectorPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCustomConnectorPluginResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin/read_updated_plugin.json")
	patchCustomConnectorPluginStub := wiremock.Patch(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins/ccp-4rrw00")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginHasBeenCreated).
		WillSetStateTo(scenarioStateCustomConnectorPluginDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedCustomConnectorPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchCustomConnectorPluginStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins/ccp-4rrw00")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginDescriptionHaveBeenUpdated).
		WillReturn(
			string(readUpdatedCustomConnectorPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedCustomConnectorPluginResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin/read_deleted_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins/ccp-4rrw00")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginHasBeenDeleted).
		WillReturn(
			string(readDeletedCustomConnectorPluginResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteCustomConnectorPluginStub := wiremock.Delete(wiremock.URLPathEqualTo("/connect/v1/custom-connector-plugins/ccp-4rrw00")).
		InScenario(customConnectorPluginScenarioName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginDescriptionHaveBeenUpdated).
		WillSetStateTo(scenarioStateCustomConnectorPluginHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteCustomConnectorPluginStub)

	customConnectorPluginDisplayName := "datagen-plugin-name"
	customConnectorPluginDescription := "datagen-plugin-description"
	// in order to test tf update (step #3)
	customConnectorPluginUpdatedDisplayName := "datagen-plugin-name-upd"
	customConnectorPluginUpdatedDescription := "datagen-plugin-description-upd"
	customConnectorPluginResourceLabel := "test_plugin_resource_label"
	fullCustomConnectorPluginResourceLabel := fmt.Sprintf("confluent_custom_connector_plugin.%s", customConnectorPluginResourceLabel)

	// Set fake values for secrets since those are required for importing
	_ = os.Setenv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_FILENAME", "foo.zip")
	defer func() {
		_ = os.Unsetenv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_FILENAME")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCustomConnectorPluginDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCustomConnectorPluginConfig(mockServerUrl, customConnectorPluginResourceLabel, customConnectorPluginDisplayName, customConnectorPluginDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginExists(fullCustomConnectorPluginResourceLabel),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "id", "ccp-4rrw00"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "display_name", customConnectorPluginDisplayName),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "description", customConnectorPluginDescription),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "cloud", "AWS"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullCustomConnectorPluginResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckCustomConnectorPluginConfig(mockServerUrl, customConnectorPluginResourceLabel, customConnectorPluginUpdatedDisplayName, customConnectorPluginUpdatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginExists(fullCustomConnectorPluginResourceLabel),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "id", "ccp-4rrw00"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "display_name", customConnectorPluginUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "description", customConnectorPluginUpdatedDescription),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "cloud", "AWS"),
				),
			},
			{
				ResourceName:      fullCustomConnectorPluginResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createCustomConnectorPluginStub, "POST /connect/v1/custom-connector-plugins", expectedCountOne)
	checkStubCount(t, wiremockClient, patchCustomConnectorPluginStub, "PATCH /connect/v1/custom-connector-plugins/ccp-4rrw00", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteCustomConnectorPluginStub, "DELETE /connect/v1/custom-connector-plugins/ccp-4rrw00", expectedCountOne)

	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckCustomConnectorPluginDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each custom connector plugin is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_service_account" {
			continue
		}
		deletedCustomConnectorPluginId := rs.Primary.ID
		req := c.ccpClient.CustomConnectorPluginsConnectV1Api.GetConnectV1CustomConnectorPlugin(c.ccpApiContext(context.Background()), deletedCustomConnectorPluginId)
		deletedCustomConnectorPlugin, response, err := req.Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			// v2/service-accounts/{nonExistentCustomConnectorPluginId/deletedCustomConnectorPluginID} returns http.StatusForbidden instead of http.StatusNotFound
			// If the error is equivalent to http.StatusNotFound, the custom connector plugin is destroyed.
			return nil
		} else if err == nil && deletedCustomConnectorPlugin.Id != nil {
			// Otherwise return the error
			if *deletedCustomConnectorPlugin.Id == rs.Primary.ID {
				return fmt.Errorf("custom connector plugin (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckCustomConnectorPluginConfig(mockServerUrl, customConnectorPluginResourceLabel, saDisplayName, saDescription string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_custom_connector_plugin" "%s" {
		display_name = "%s"
		description = "%s"
		documentation_link = "https://www.confluent.io/hub/confluentinc/kafka-connect-datagen"
		connector_class = "io.confluent.kafka.connect.datagen.DatagenConnector"
		connector_type = "SOURCE"
		sensitive_config_properties = ["keys", "passwords"]
		filename = "foo.zip"
	}
	`, mockServerUrl, customConnectorPluginResourceLabel, saDisplayName, saDescription)
}

func testAccCheckCustomConnectorPluginExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s custom connector plugin has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s custom connector plugin", n)
		}

		return nil
	}
}
