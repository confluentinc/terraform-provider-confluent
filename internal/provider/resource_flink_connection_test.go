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
	scenarioStateConnectionHasBeenCreated = "A new connection has been just created"
	scenarioStateConnectionHasBeenDeleted = "The connection has been deleted"
	connectionScenarioName                = "confluent_flink_connection Resource Lifecycle"

	flinkConnectionDisplayName = "Connection1"
	flinkConnectionNameTest    = "connection-test"
	flinkConnectionApiVersion  = "sql/v1"
	flinkConnectionKind        = "Connection"
	flinkConnectionType        = "OPENAI"
	flinkEndpoint              = "https://api.openai.com/v1/chat/completions"
	flinkAPIKey                = "OPENAI"
)

var createFlinkConnectionPath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/connections", flinkOrganizationIdTest, flinkEnvironmentIdTest)
var readFlinkConnectionPath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/connections/%s", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkConnectionDisplayName)

func TestAccFlinkConnection(t *testing.T) {
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
	_ = os.Setenv("IMPORT_FLINK_API_KEY", kafkaApiKey)
	_ = os.Setenv("IMPORT_FLINK_API_SECRET", kafkaApiSecret)
	_ = os.Setenv("IMPORT_FLINK_REST_ENDPOINT", mockTestServerUrl)
	_ = os.Setenv("IMPORT_FLINK_PRINCIPAL_ID", flinkPrincipalIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ORGANIZATION_ID", flinkOrganizationIdTest)
	_ = os.Setenv("IMPORT_CONFLUENT_ENVIRONMENT_ID", flinkEnvironmentIdTest)
	_ = os.Setenv("IMPORT_FLINK_COMPUTE_POOL_ID", flinkComputePoolIdTest)
	defer func() {
		_ = os.Unsetenv("API_KEY")
		_ = os.Unsetenv("IMPORT_FLINK_API_KEY")
		_ = os.Unsetenv("IMPORT_FLINK_API_SECRET")
		_ = os.Unsetenv("IMPORT_FLINK_REST_ENDPOINT")
		_ = os.Unsetenv("IMPORT_FLINK_PRINCIPAL_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ORGANIZATION_ID")
		_ = os.Unsetenv("IMPORT_CONFLUENT_ENVIRONMENT_ID")
		_ = os.Unsetenv("IMPORT_FLINK_COMPUTE_POOL_ID")
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
				Config: testAccCheckConnectionConfig(mockTestServerUrl, flinkConnectionResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectionExists(fullConnectionResourceLabel),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramDisplayName, flinkConnectionDisplayName),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramType, flinkConnectionType),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramEndpoint, flinkEndpoint),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramApiKey, flinkAPIKey),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramApiVersion, flinkConnectionApiVersion),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramKind, flinkConnectionKind),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkEnvironmentIdTest),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramOrganization, paramId), flinkOrganizationIdTest),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramComputePool, paramId), flinkComputePoolIdTest),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, fmt.Sprintf("%s.0.%s", paramPrincipal, paramId), flinkPrincipalIdTest),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, paramRestEndpoint, mockTestServerUrl),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, "credentials.0.key", kafkaApiKey),
					resource.TestCheckResourceAttr(fullConnectionResourceLabel, "credentials.0.secret", kafkaApiSecret),
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

func testAccCheckConnectionConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
    	endpoint = "%s"
	}

	resource "confluent_flink_connection" "%s" {
      credentials {
        key = "%s"
        secret = "%s"
      }
      rest_endpoint = "%s"
      principal {
         id = "%s"
      }
      organization {
         id = "%s"
      }
      environment {
         id = "%s"
      }
      compute_pool {
         id = "%s"
      }
      display_name  = "%s"
	  type          = "%s"
	  endpoint      = "%s"
 	  api_key 		= "%s"
	}
	`, mockServerUrl, resourceLabel, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkConnectionDisplayName, flinkConnectionType, flinkEndpoint, flinkAPIKey)
}

func testAccCheckConnectionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s connection has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s connection", n)
		}

		return nil
	}
}

func testAccCheckConnectionDestroy(s *terraform.State, url string) error {
	testClient := testAccProvider.Meta().(*Client)
	c := testClient.flinkRestClientFactory.CreateFlinkRestClient(url, flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkPrincipalIdTest, kafkaApiKey, kafkaApiSecret, false, testClient.oauthToken)
	// Loop through the resources in state, verifying each Kafka topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_connection" {
			continue
		}
		deletedTopicId := rs.Primary.ID
		_, response, err := c.apiClient.ConnectionsSqlV1Api.GetSqlv1Connection(c.apiContext(context.Background()), flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkConnectionNameTest).Execute()
		if response != nil && (response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound) {
			return nil
		} else if err == nil && deletedTopicId != "" {
			// Otherwise return the error
			if deletedTopicId == rs.Primary.ID {
				return fmt.Errorf("topic (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}
