package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	CatalogIntegrationDataSourceScenarioName = "confluent_catalog_integration Data Source Lifecycle"
)

func TestAccDataSourceCatalogIntegration(t *testing.T) {
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

	readCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/read_created_aws_glue_ci.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/tableflow/v1/catalog-integrations/tci-abc123")).
		InScenario(CatalogIntegrationDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	CatalogIntegrationResourceName := "data.confluent_catalog_integration.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceCatalogIntegration(mockServerUrl, "tci-abc123"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "id", "tci-abc123"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "display_name", "catalog_integration_1"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "environment.#", "1"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "suspended", "false"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "aws_glue.#", "1"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "snowflake.#", "0"),
					resource.TestCheckResourceAttr(CatalogIntegrationResourceName, "aws_glue.0.provider_integration_id", "cspi-stgce89r7"),
				),
			},
		},
	})

}

func testAccCheckDataSourceCatalogIntegration(mockServerUrl, resourceId string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	data "confluent_catalog_integration" "main" {
	    id = "%s"
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, resourceId)
}
