package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

const (
	scenarioStatePluginHasBeenCreated = "The plugin has been just created"
	scenarioStatePluginHasBeenUpdated = "The plugin has been just updated"
	scenarioStatePluginHasBeenDeleted = "The  plugin has been deleted"
	pluginScenarioName                = "confluent_plugin Resource Lifecycle"
	pluginEnvironment                 = "env-123"
)

func TestAccPlugin(t *testing.T) {
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

	createPluginResponse, _ := ioutil.ReadFile("../testdata/plugin/create_plugin.json")
	createPluginStub := wiremock.Post(wiremock.URLPathEqualTo("/ccpm/v1/plugins")).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStatePluginHasBeenCreated).
		WillReturn(
			string(createPluginResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createPluginStub)

	readCreatedPluginResponse, _ := ioutil.ReadFile("../testdata/plugin/read_created_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins/ccp-devczmp7p7")).
		WithQueryParam("environment", wiremock.EqualTo(pluginEnvironment)).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(scenarioStatePluginHasBeenCreated).
		WillReturn(
			string(readCreatedPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedPluginResponse, _ := ioutil.ReadFile("../testdata/plugin/read_updated_plugin.json")
	patchPluginStub := wiremock.Patch(wiremock.URLPathEqualTo("/ccpm/v1/plugins/ccp-devczmp7p7")).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(scenarioStatePluginHasBeenCreated).
		WillSetStateTo(scenarioStatePluginHasBeenUpdated).
		WillReturn(
			string(readUpdatedPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchPluginStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins/ccp-devczmp7p7")).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(scenarioStatePluginHasBeenUpdated).
		WillReturn(
			string(readUpdatedPluginResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deletePluginStub := wiremock.Delete(wiremock.URLPathEqualTo("/ccpm/v1/plugins/ccp-devczmp7p7")).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(scenarioStatePluginHasBeenUpdated).
		WillSetStateTo(scenarioStatePluginHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deletePluginStub)

	readDeletedPluginResponse, _ := ioutil.ReadFile("../testdata/plugin/read_deleted_plugin.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/ccpm/v1/plugins/ccp-devczmp7p7")).
		WithQueryParam("environment", wiremock.EqualTo(pluginEnvironment)).
		InScenario(pluginScenarioName).
		WhenScenarioStateIs(scenarioStatePluginHasBeenDeleted).
		WillReturn(
			string(readDeletedPluginResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	pluginDisplayName := "plugin-name"
	pluginDescription := "plugin-description"
	// in order to test tf update (step #3)
	pluginUpdatedDisplayName := "plugin-name-upd"
	pluginUpdatedDescription := "plugin-description-upd"
	pluginResourceLabel := "test"
	fullPluginResourceLabel := fmt.Sprintf("confluent_plugin.%s", pluginResourceLabel)

	_ = os.Setenv("IMPORT_ENVIRONMENT", "env-123")
	defer func() {
		_ = os.Unsetenv("IMPORT_ENVIRONMENT")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccPluginConfig(mockServerUrl, pluginResourceLabel, pluginDisplayName, pluginDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginExists(fullPluginResourceLabel),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "id", "ccp-devczmp7p7"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "display_name", pluginDisplayName),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "description", pluginDescription),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "cloud", "AWS"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "runtime_language", "JAVA"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "environment.0.id", "env-123"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullPluginResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPluginConfig(mockServerUrl, pluginResourceLabel, pluginUpdatedDisplayName, pluginUpdatedDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPluginExists(fullPluginResourceLabel),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "id", "ccp-devczmp7p7"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "display_name", pluginUpdatedDisplayName),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "description", pluginUpdatedDescription),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "cloud", "AWS"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "runtime_language", "JAVA"),
					resource.TestCheckResourceAttr(fullPluginResourceLabel, "environment.0.id", "env-123"),
				),
			},
			{
				ResourceName:      fullPluginResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createPluginStub, "POST /connect/v1/plugins", expectedCountOne)
	checkStubCount(t, wiremockClient, patchPluginStub, "PATCH /connect/v1/plugins/ccp-devczmp7p7", expectedCountOne)
	checkStubCount(t, wiremockClient, deletePluginStub, "DELETE /connect/v1/plugins/ccp-devczmp7p7", expectedCountOne)
}

func testAccPluginConfig(mockServerUrl, pluginResourceLabel, displayName, description string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_plugin" "%s" {
		display_name = "%s"
		description = "%s"
		cloud = "AWS"
    	environment {
          id = "env-123"
        }
	}
	`, mockServerUrl, pluginResourceLabel, displayName, description)
}

func testAccCheckPluginExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s plugin has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s plugin", n)
		}

		return nil
	}
}
