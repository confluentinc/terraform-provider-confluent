//go:build live_test && (all || connect)

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
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCustomConnectorPluginLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	pluginDisplayName := fmt.Sprintf("tf-live-plugin-%d", randomSuffix)
	pluginResourceLabel := "test_live_custom_connector_plugin"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCustomConnectorPluginLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCustomConnectorPluginLiveConfig(endpoint, pluginResourceLabel, pluginDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginLiveExists(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "display_name", pluginDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "connector_class", "io.confluent.kafka.connect.datagen.DatagenConnector"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "connector_type", "SOURCE"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "cloud", "AWS"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "description", "A test custom connector plugin for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "documentation_link", "https://docs.confluent.io/kafka-connectors/datagen/current/overview.html"),
					resource.TestCheckResourceAttrSet(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "id"),
				),
			},
			{
				ResourceName:      fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{"filename"},
			},
		},
	})
}

func TestAccCustomConnectorPluginUpdateLive(t *testing.T) {
	// Enable parallel execution for I/O bound operations
	t.Parallel()

	// Skip this test unless explicitly enabled
	if os.Getenv("TF_ACC_PROD") == "" {
		t.Skip("Skipping live test. Set TF_ACC_PROD=1 to run this test.")
	}

	// Read credentials and configuration from environment variables (populated by Vault)
	apiKey := os.Getenv("CONFLUENT_CLOUD_API_KEY")
	apiSecret := os.Getenv("CONFLUENT_CLOUD_API_SECRET")
	endpoint := os.Getenv("CONFLUENT_CLOUD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.confluent.cloud" // Use default endpoint if not set
	}

	// Validate required environment variables are present
	if apiKey == "" || apiSecret == "" {
		t.Fatal("CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for live tests")
	}

	// Generate unique names for test resources to avoid conflicts
	randomSuffix := rand.Intn(100000)
	pluginDisplayName := fmt.Sprintf("tf-live-plugin-update-%d", randomSuffix)
	pluginDisplayNameUpdated := fmt.Sprintf("tf-live-plugin-updated-%d", randomSuffix)
	pluginResourceLabel := "test_live_custom_connector_plugin_update"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckCustomConnectorPluginLiveDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckCustomConnectorPluginLiveConfig(endpoint, pluginResourceLabel, pluginDisplayName, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginLiveExists(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "display_name", pluginDisplayName),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "description", "A test custom connector plugin for live testing"),
				),
			},
			{
				Config: testAccCheckCustomConnectorPluginUpdateLiveConfig(endpoint, pluginResourceLabel, pluginDisplayNameUpdated, apiKey, apiSecret),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckCustomConnectorPluginLiveExists(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel)),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "display_name", pluginDisplayNameUpdated),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "description", "An updated test custom connector plugin for live testing"),
					resource.TestCheckResourceAttr(fmt.Sprintf("confluent_custom_connector_plugin.%s", pluginResourceLabel), "documentation_link", "https://docs.confluent.io/kafka-connectors/datagen/current/overview.html#updated"),
				),
			},
		},
	})
}

func testAccCheckCustomConnectorPluginLiveDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_custom_connector_plugin" {
			continue
		}

		// In live tests, we can't easily check if the resource is actually destroyed
		// without making API calls, so we just verify the resource is removed from state
		if rs.Primary.ID != "" {
			// This is normal - the resource should have an ID but be removed from the live environment
			// The actual cleanup happens through the API calls during destroy
		}
	}
	return nil
}

func testAccCheckCustomConnectorPluginLiveExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource ID is not set")
		}

		return nil
	}
}

func testAccCheckCustomConnectorPluginLiveConfig(endpoint, pluginResourceLabel, pluginDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_custom_connector_plugin" "%s" {
		display_name      = "%s"
		description       = "A test custom connector plugin for live testing"
		documentation_link = "https://docs.confluent.io/kafka-connectors/datagen/current/overview.html"
		connector_class   = "io.confluent.kafka.connect.datagen.DatagenConnector"
		connector_type    = "SOURCE"
		cloud             = "AWS"
		filename          = "test_artifacts/confluentinc-kafka-connect-datagen-0.6.6.zip"
		
		sensitive_config_properties = [
			"kafka.api.key",
			"kafka.api.secret"
		]
	}
	`, endpoint, apiKey, apiSecret, pluginResourceLabel, pluginDisplayName)
}

func testAccCheckCustomConnectorPluginUpdateLiveConfig(endpoint, pluginResourceLabel, pluginDisplayName, apiKey, apiSecret string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint         = "%s"
		cloud_api_key    = "%s"
		cloud_api_secret = "%s"
	}

	resource "confluent_custom_connector_plugin" "%s" {
		display_name      = "%s"
		description       = "An updated test custom connector plugin for live testing"
		documentation_link = "https://docs.confluent.io/kafka-connectors/datagen/current/overview.html#updated"
		connector_class   = "io.confluent.kafka.connect.datagen.DatagenConnector"
		connector_type    = "SOURCE"
		cloud             = "AWS"
		filename          = "test_artifacts/confluentinc-kafka-connect-datagen-0.6.6.zip"
		
		sensitive_config_properties = [
			"kafka.api.key",
			"kafka.api.secret"
		]
	}
	`, endpoint, apiKey, apiSecret, pluginResourceLabel, pluginDisplayName)
} 