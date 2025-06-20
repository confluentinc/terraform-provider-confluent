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
	scenarioStateCustomConnectorPluginVersionPresignedUrlHasBeenCreated = "The new custom connector plugin version's presigned URL has been just created"
	scenarioStateCustomConnectorPluginVersionHasBeenCreated             = "The new custom connector plugin version has been just created"
	scenarioStateCustomConnectorPluginVersionHasBeenDeleted             = "The new custom connector plugin version has been deleted"
	customConnectorPluginScenarioVersionName                            = "confluent_custom_connector_plugin_version Resource Lifecycle"
)

func TestAccCustomConnectorPluginVersion(t *testing.T) {
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
	createCustomConnectorPluginVersionPresignedUrlResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin_version/read_presigned_url.json")
	createCustomConnectorPluginPresignedVersionUrlStub := wiremock.Post(wiremock.URLPathEqualTo("/ccpm/v1/presigned-upload-url")).
		InScenario(customConnectorPluginScenarioVersionName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCustomConnectorPluginVersionPresignedUrlHasBeenCreated).
		WillReturn(
			string(createCustomConnectorPluginVersionPresignedUrlResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCustomConnectorPluginPresignedVersionUrlStub)

	createCustomConnectorPluginVersionResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin_version/create_plugin.json")
	createCustomConnectorPluginStub := wiremock.Post(wiremock.URLPathEqualTo("/ccpm/v1/plugins/foo/versions")).
		InScenario(customConnectorPluginScenarioVersionName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginVersionPresignedUrlHasBeenCreated).
		WillSetStateTo(scenarioStateCustomConnectorPluginVersionHasBeenCreated).
		WillReturn(
			string(createCustomConnectorPluginVersionResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCustomConnectorPluginStub)

	readCreatedCustomConnectorPluginVersionResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin_version/read_created_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins/foo/versions/dlz-f3a90de")).
		WithQueryParam("environment", wiremock.EqualTo("env-00000")).
		InScenario(customConnectorPluginScenarioVersionName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginVersionHasBeenCreated).
		WillReturn(
			string(readCreatedCustomConnectorPluginVersionResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedCustomConnectorPluginVersionResponse, _ := ioutil.ReadFile("../testdata/custom_connector_plugin_version/read_deleted_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins/foo/versions/dlz-f3a90de")).
		InScenario(customConnectorPluginScenarioVersionName).
		WithQueryParam("environment", wiremock.EqualTo("env-00000")).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginVersionHasBeenDeleted).
		WillReturn(
			string(readDeletedCustomConnectorPluginVersionResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteCustomConnectorPluginStub := wiremock.Delete(wiremock.URLPathEqualTo("/ccpm/v1/plugins/foo/versions/dlz-f3a90de")).
		InScenario(customConnectorPluginScenarioVersionName).
		WhenScenarioStateIs(scenarioStateCustomConnectorPluginVersionHasBeenCreated).
		WillSetStateTo(scenarioStateCustomConnectorPluginVersionHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteCustomConnectorPluginStub)

	customConnectorPluginResourceLabel := "test"
	fullCustomConnectorPluginResourceLabel := fmt.Sprintf("confluent_custom_connector_plugin_version.%s", customConnectorPluginResourceLabel)

	_ = os.Setenv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_VERSION_FILENAME", "foo.zip")
	defer func() {
		_ = os.Unsetenv("IMPORT_CUSTOM_CONNECTOR_PLUGIN_VERSION_FILENAME")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCustomConnectorPluginVersionConfig(mockServerUrl, customConnectorPluginResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginVersionExists(fullCustomConnectorPluginResourceLabel),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "id", "dlz-f3a90de"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "cloud", "AWS"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "filename", "foo.zip"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "plugin_id", "foo"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "version", "v0.0.1"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "documentation_link", "https://github.com/confluentinc/kafka-connect-datagen"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "sensitive_config_properties.#", "3"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "sensitive_config_properties.0", "keys"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "connector_class.#", "1"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "connector_class.0.%", "2"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "connector_class.0.connector_class_name", "io.confluent.kafka.connect.datagen.DatagenConnector"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "connector_class.0.connector_type", "SOURCE"),
					resource.TestCheckResourceAttr(fullCustomConnectorPluginResourceLabel, "environment.0.id", "env-00000"),
				),
			},
		},
	})

	checkStubCount(t, wiremockClient, createCustomConnectorPluginStub, "POST /connect/v1/custom-connector-plugins", expectedCountOne)
	checkStubCount(t, wiremockClient, deleteCustomConnectorPluginStub, "DELETE /connect/v1/custom-connector-plugins/ccp-4rrw00", expectedCountOne)
}

func testAccCheckCustomConnectorPluginVersionConfig(mockServerUrl, customConnectorPluginResourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_custom_connector_plugin_version" "%s" {
      version = "v0.0.1"
      cloud = "AWS"
      documentation_link          = "https://github.com/confluentinc/kafka-connect-datagen"
      connector_class {
          connector_class_name    = "io.confluent.kafka.connect.datagen.DatagenConnector"
          connector_type          = "SOURCE"
        }
      sensitive_config_properties = [ "passwords", "keys", "tokens"]
      filename                    = "foo.zip"
      plugin_id 				  = "foo"
      environment {
          id = "env-00000"
        }
	}
	`, mockServerUrl, customConnectorPluginResourceLabel)
}

func testAccCheckCustomConnectorPluginVersionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s custom connector plugin version has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s custom connector pluginversion", n)
		}

		return nil
	}
}
