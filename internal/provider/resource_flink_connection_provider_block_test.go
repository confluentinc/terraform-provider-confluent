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

func TestAccFlinkConnectionProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockTestServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createFlinkConnectionResponse, _ := ioutil.ReadFile("../testdata/flink_connection/create_connection.json")
	createFlinkConnectionStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkConnectionPath)).
		InScenario(connectionScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateConnectionHasBeenCreated).
		WillReturn(
			string(createFlinkConnectionResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkConnectionStub)

	readCreatedConnectionsResponse, _ := ioutil.ReadFile("../testdata/flink_connection/read_connection.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkConnectionPath)).
		InScenario(connectionScenarioName).
		WhenScenarioStateIs(scenarioStateConnectionHasBeenCreated).
		WillReturn(
			string(readCreatedConnectionsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteConnectionStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkConnectionPath)).
		InScenario(connectionScenarioName).
		WhenScenarioStateIs(scenarioStateConnectionHasBeenCreated).
		WillSetStateTo(scenarioStateConnectionHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteConnectionStub)

	readDeletedConnectionResponse, _ := ioutil.ReadFile("../testdata/flink_connection/read_deleted_connection.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkConnectionPath)).
		InScenario(connectionScenarioName).
		WhenScenarioStateIs(scenarioStateConnectionHasBeenDeleted).
		WillReturn(
			string(readDeletedConnectionResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	flinkConnectionResourceLabel := "test"
	fullConnectionResourceLabel := fmt.Sprintf("confluent_flink_connection.%s", flinkConnectionResourceLabel)

	_ = os.Setenv("API_KEY", flinkAPIKey)
	defer func() {
		_ = os.Unsetenv("API_KEY")
	}()
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckConnectionDestroy(s, mockTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectionConfigProviderBlock(mockTestServerUrl, flinkConnectionResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(fullConnectionResourceLabel),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramDisplayName, flinkConnectionDisplayName),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramType, flinkConnectionType),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramEndpoint, flinkEndpoint),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramApiKey, flinkAPIKey),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramApiVersion, flinkConnectionApiVersion),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramKind, flinkConnectionKind),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId)),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, paramRestEndpoint),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullConnectionResourceLabel, "credentials.0.secret"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullConnectionResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fullConnectionResourceLabel].Primary.ID
					return resourceId, nil
				},
			},
		},
	})
}

func testAccCheckConnectionConfigProviderBlock(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
		flink_api_key         = "%s"
		flink_api_secret      = "%s"
		flink_rest_endpoint   = "%s"
		flink_principal_id    = "%s" 
		organization_id       = "%s"
		environment_id        = "%s"
		flink_compute_pool_id ="%s"
	}
	resource "confluent_flink_connection" "%s" {
        display_name     = "%s"
        type            = "%s"
	    endpoint           = "%s"
	    api_key 		= "%s"
	}
	`, mockServerUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, resourceLabel, flinkConnectionDisplayName, flinkConnectionType, flinkEndpoint, flinkAPIKey)
}
