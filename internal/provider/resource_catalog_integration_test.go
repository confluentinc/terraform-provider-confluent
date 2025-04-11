// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateCatalogIntegrationIsProvisioning = "The new catalog integration is provisioning"
	scenarioStateCatalogIntegrationHasBeenCreated = "The new catalog integration has been just created"
	scenarioStateCatalogIntegrationHasBeenUpdated = "The new catalog integration has been updated"
	byobAwsCatalogIntegrationScenarioName         = "confluent_catalog_integration Byob Aws Resource Lifecycle"
	snowflakeCatalogIntegrationScenarioName       = "confluent_catalog_integration Snowflake Resource Lifecycle"

	catalogIntegrationUrlPath       = "/tableflow/v1/catalog-integrations"
	catalogIntegrationResourceLabel = "confluent_catalog_integration.main"
)

func TestAccCatalogIntegrationAwsGlue(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	createCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/create_aws_glue_ci.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(catalogIntegrationUrlPath)).
		InScenario(byobAwsCatalogIntegrationScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		//WillSetStateTo(scenarioStateCatalogIntegrationIsProvisioning).
		WillSetStateTo(scenarioStateCatalogIntegrationHasBeenCreated).
		WillReturn(
			string(createCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	catalogIntegrationReadUrlPath := fmt.Sprintf("%s/tci-abc123", catalogIntegrationUrlPath)
	/*_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
	InScenario(byobAwsCatalogIntegrationScenarioName).
	WhenScenarioStateIs(scenarioStateCatalogIntegrationIsProvisioning).
	WillSetStateTo(scenarioStateCatalogIntegrationHasBeenCreated).
	WillReturn(
		string(createCatalogIntegrationResponse),
		contentTypeJSONHeader,
		http.StatusOK,
	))*/

	readCreatedCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/read_created_aws_glue_ci.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(byobAwsCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenCreated).
		WillReturn(
			string(readCreatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/update_aws_glue_ci.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(byobAwsCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenCreated).
		WillSetStateTo(scenarioStateCatalogIntegrationHasBeenUpdated).
		WillReturn(
			string(updatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(byobAwsCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenUpdated).
		WillReturn(
			string(updatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(byobAwsCatalogIntegrationScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceCatalogIntegrationAwsGlue(mockServerUrl, "catalog_integration_1"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "id", "tci-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "display_name", "catalog_integration_1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.#", "0"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.0.provider_integration_id", "cspi-stgce89r7"),
				),
			},
			{
				Config: testAccCheckResourceCatalogIntegrationAwsGlue(mockServerUrl, "catalog_integration_2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "id", "tci-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "display_name", "catalog_integration_2"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.#", "0"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.0.provider_integration_id", "cspi-stgce89r7"),
				),
			},
		},
	})
}

func TestAccCatalogIntegrationSnowflake(t *testing.T) {
	ctx := context.Background()

	time.Sleep(5 * time.Second)
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

	createCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/create_snowflake_ci.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(catalogIntegrationUrlPath)).
		InScenario(snowflakeCatalogIntegrationScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		//WillSetStateTo(scenarioStateCatalogIntegrationIsProvisioning).
		WillSetStateTo(scenarioStateCatalogIntegrationHasBeenCreated).
		WillReturn(
			string(createCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	catalogIntegrationReadUrlPath := fmt.Sprintf("%s/tci-abc123", catalogIntegrationUrlPath)
	/*_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
	InScenario(snowflakeCatalogIntegrationScenarioName).
	WhenScenarioStateIs(scenarioStateCatalogIntegrationIsProvisioning).
	WillSetStateTo(scenarioStateCatalogIntegrationHasBeenCreated).
	WillReturn(
		string(createCatalogIntegrationResponse),
		contentTypeJSONHeader,
		http.StatusOK,
	))*/

	readCreatedCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/read_created_snowflake_ci.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(snowflakeCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenCreated).
		WillReturn(
			string(readCreatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updatedCatalogIntegrationResponse, _ := os.ReadFile("../testdata/catalog_integration/update_snowflake_ci.json")
	_ = wiremockClient.StubFor(wiremock.Patch(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(snowflakeCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenCreated).
		WillSetStateTo(scenarioStateCatalogIntegrationHasBeenUpdated).
		WillReturn(
			string(updatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(snowflakeCatalogIntegrationScenarioName).
		WhenScenarioStateIs(scenarioStateCatalogIntegrationHasBeenUpdated).
		WillReturn(
			string(updatedCatalogIntegrationResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(catalogIntegrationReadUrlPath)).
		InScenario(snowflakeCatalogIntegrationScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckResourceCatalogIntegrationSnowflake(mockServerUrl, "catalog_integration_1", "https://vuser1_polaris.snowflakecomputing.com/", "client-id", "client-secret", "warehouse-name", "allowed-scope"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "id", "tci-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "display_name", "catalog_integration_1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.#", "0"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.endpoint", "https://vuser1_polaris.snowflakecomputing.com/"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.warehouse", "warehouse-name"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.allowed_scope", "allowed-scope"),
				),
			},
			{
				Config: testAccCheckResourceCatalogIntegrationSnowflake(mockServerUrl, "catalog_integration_2", "https://vuser2_polaris.snowflakecomputing.com/", "client-id-2", "client-secret-2", "warehouse-name-2", "allowed-scope-2"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "id", "tci-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "display_name", "catalog_integration_2"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "kafka_cluster.0.id", "lkc-00000"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "suspended", "false"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "aws_glue.#", "0"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.#", "1"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.endpoint", "https://vuser2_polaris.snowflakecomputing.com/"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.warehouse", "warehouse-name-2"),
					resource.TestCheckResourceAttr(catalogIntegrationResourceLabel, "snowflake.0.allowed_scope", "allowed-scope-2"),
				),
			},
		},
	})
}

func testAccCheckResourceCatalogIntegrationAwsGlue(mockServerUrl, display_name string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_catalog_integration" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		aws_glue {
			provider_integration_id = "cspi-stgce89r7"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, display_name)
}

func testAccCheckResourceCatalogIntegrationSnowflake(mockServerUrl, displayName, endpoint, clientId, clientSecret, warehouse, allowedScope string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	resource "confluent_catalog_integration" "main" {
		display_name = "%s"
		environment {
			id = "env-abc123"
		}
		kafka_cluster {
			id = "lkc-00000"
		}
		snowflake {
			endpoint = "%s"
			client_id = "%s"
			client_secret = "%s"
			warehouse = "%s"
			allowed_scope = "%s"
		}
		credentials {
			key = "test_key"
			secret = "test_secret"
		}
	}
	`, mockServerUrl, displayName, endpoint, clientId, clientSecret, warehouse, allowedScope)
}
