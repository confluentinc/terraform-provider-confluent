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
	scenarioFlinkComputePoolConfigHasBeenCreated = "A new compute pool config has been just created"
	scenarioFlinkComputePoolConfigHasBeenUpdated = "The compute pool config has been updated"
	scenarioFlinkComputePoolConfigHasBeenDeleted = "The compute pool config has been deleted"
	flinkComputePoolConfigScenarioName           = "confluent_flink_compute_pool_config Resource Lifecycle"

	testFlinkComputePoolConfigResourceLabel = "test_flink_compute_pool_resource_label"
)

var fullFlinkComputePoolConfigResourceLabel = fmt.Sprintf("confluent_flink_compute_pool_config.%s", testFlinkComputePoolConfigResourceLabel)

func TestAccComputePoolConfig(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockFlinkComputePoolConfigTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := mockFlinkComputePoolConfigTestServerUrl
	wiremockClient := wiremock.NewClient(mockFlinkComputePoolConfigTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	createFlinkComputePoolConfigResponse, _ := ioutil.ReadFile("../testdata/flink_compute_pool_config/read_created_compute_pool_config.json")
	createFlinkComputePoolConfigStub := wiremock.Patch(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolConfigScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioFlinkComputePoolConfigHasBeenCreated).
		WillReturn(
			string(createFlinkComputePoolConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(createFlinkComputePoolConfigStub)

	readCreatedComputePoolConfigResponse, _ := ioutil.ReadFile("../testdata/flink_compute_pool_config/read_created_compute_pool_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolConfigScenarioName).
		WhenScenarioStateIs(scenarioFlinkComputePoolConfigHasBeenCreated).
		WillReturn(
			string(readCreatedComputePoolConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateComputePoolConfigResponse, _ := ioutil.ReadFile("../testdata/flink_compute_pool_config/read_updated_compute_pool_config.json")
	updateComputePoolConfigStub := wiremock.Patch(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolConfigScenarioName).
		WhenScenarioStateIs(scenarioFlinkComputePoolConfigHasBeenCreated).
		WillSetStateTo(scenarioFlinkComputePoolConfigHasBeenUpdated).
		WillReturn(
			string(updateComputePoolConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(updateComputePoolConfigStub)

	readUpdatedComputePoolConfigResponse, _ := ioutil.ReadFile("../testdata/flink_compute_pool_config/read_updated_compute_pool_config.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolConfigScenarioName).
		WhenScenarioStateIs(scenarioFlinkComputePoolConfigHasBeenUpdated).
		WillReturn(
			string(readUpdatedComputePoolConfigResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteComputePoolConfigStub := wiremock.Delete(wiremock.URLPathEqualTo("/fcpm/v2/compute-pool-config")).
		InScenario(flinkComputePoolConfigScenarioName).
		WhenScenarioStateIs(scenarioFlinkComputePoolConfigHasBeenUpdated).
		WillSetStateTo(scenarioFlinkComputePoolConfigHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(deleteComputePoolConfigStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		//CheckDestroy:      testAccCheckSchemaRegistryClusterModeDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkComputePoolConfig(confluentCloudBaseUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolConfigExists(fullFlinkComputePoolConfigResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "default_compute_pool_enabled", "true"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "default_max_cfu", "10"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "api_version", "fcpm/v2"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "kind", "OrgComputePoolConfig"),
				),
			},
			{
				Config: testAccCheckFlinkComputePoolConfigUpdated(confluentCloudBaseUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputePoolConfigExists(fullFlinkComputePoolConfigResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "default_compute_pool_enabled", "false"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "default_max_cfu", "15"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "api_version", "fcpm/v2"),
					resource.TestCheckResourceAttr(fullFlinkComputePoolConfigResourceLabel, "kind", "OrgComputePoolConfig"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkComputePoolConfigResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckFlinkComputePoolConfig(confluentCloudBaseUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_flink_compute_pool_config" "%s" {
	  default_compute_pool_enabled = true
     default_max_cfu = 10

	}
	`, confluentCloudBaseUrl, testFlinkComputePoolConfigResourceLabel)
}

func testAccCheckFlinkComputePoolConfigUpdated(confluentCloudBaseUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_flink_compute_pool_config" "%s" {
	  default_compute_pool_enabled = false
      default_max_cfu = 15

	}
	`, confluentCloudBaseUrl, testFlinkComputePoolConfigResourceLabel)
}
