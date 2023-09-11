// Copyright 2023 Confluent Inc. All Rights Reserved.
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	schemaExporterResourceScenarioName        = "confluent_schema_exporter Resource Lifecycle"
	scenarioStateSchemaExporterHasBeenCreated = "A new schema exporter has been just created"
	scenarioStateSchemaExporterHasBeenUpdated = "A new schema exporter has been just updated"
	createSchemaExporterUrlPath               = "/exporters"
	readCreatedSchemaExporterUrlPath          = "/exporters/exporter1"
	readCreatedSchemaExporterStatusUrlPath    = "/exporters/exporter1/status"
	schemaExporterLabel                       = "confluent_schema_exporter.main"
)

func TestAccSchemaExporter(t *testing.T) {
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

	generalResponse, _ := ioutil.ReadFile("../testdata/schema_exporter/general_response.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	createdExporter, _ := ioutil.ReadFile("../testdata/schema_exporter/created_exporter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(createdExporter),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WillSetStateTo(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	updatedExporter, _ := ioutil.ReadFile("../testdata/schema_exporter/updated_exporter.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(updatedExporter),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/pause")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/resume")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath+"/reset")).
		InScenario(schemaExporterResourceScenarioName).
		WillReturn(
			string(generalResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	runningStatusResponse, _ := ioutil.ReadFile("../testdata/schema_exporter/running_status.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterStatusUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenCreated).
		WillReturn(
			string(runningStatusResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	pausedStatusResponse, _ := ioutil.ReadFile("../testdata/schema_exporter/pause_status.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedSchemaExporterStatusUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
		WhenScenarioStateIs(scenarioStateSchemaExporterHasBeenUpdated).
		WillReturn(
			string(pausedStatusResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(readCreatedSchemaExporterUrlPath)).
		InScenario(schemaExporterResourceScenarioName).
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
				Config: schemaExporterResourceConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(schemaExporterLabel, "name", "exporter1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context", "tc"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context_type", "CUSTOM"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "status", "RUNNING"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.0", "foo"),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.%%", paramConfigs), "0"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.rest_endpoint", "https://psrc-4xgzx.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.key", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.secret", "11"),
				),
			},
			{
				Config: schemaExporterResourceUpdatedConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(schemaExporterLabel, "name", "exporter1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context", "tc-3"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "context_type", "CUSTOM"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "status", "PAUSED"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "subjects.0", "foo3"),
					resource.TestCheckResourceAttr(schemaExporterLabel, fmt.Sprintf("%s.%%", paramConfigs), "0"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.rest_endpoint", "https://psrc-5xgzx.us-east-2.aws.confluent.cloud"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.#", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.key", "1"),
					resource.TestCheckResourceAttr(schemaExporterLabel, "destination_schema_registry_cluster.0.credentials.0.secret", "12"),
				),
			},
		},
	})
}

func schemaExporterResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	    schema_registry_id = "xxx"
	    schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	    schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	    schema_registry_api_secret = "x"
 	}
 	resource "confluent_schema_exporter" "main" {
		name = "exporter1"
		context = "tc"
		context_type = "CUSTOM"    
		subjects = ["foo"]

		destination_schema_registry_cluster {
		  rest_endpoint = "https://psrc-4xgzx.us-east-2.aws.confluent.cloud"
		  basic_auth_credentials_source = "USER_INFO"
		  credentials {
			key    = "1"
			secret = "11"
		  }
		}
	}

 	`, mockServerUrl)
}

func schemaExporterResourceUpdatedConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	    schema_registry_id = "xxx"
	    schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	    schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	    schema_registry_api_secret = "x"
 	}
 	resource "confluent_schema_exporter" "main" {
	    name = "exporter1"
		context = "tc-3"
		context_type = "CUSTOM"    
		subjects = ["foo3"]

        status = "PAUSED"

		destination_schema_registry_cluster {
		  rest_endpoint = "https://psrc-5xgzx.us-east-2.aws.confluent.cloud"
		  basic_auth_credentials_source = "USER_INFO"
		  credentials {
			key    = "1"
			secret = "12"
		  }
		}
	}
 	`, mockServerUrl)
}
