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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	scenarioStateStatementHasBeenCreated   = "A new statement has been just created"
	scenarioStateStatementIsPending        = "A new statement is pending"
	scenarioStateStatementIsDeleting       = "The statement is being deleted"
	scenarioStateStatementIsResuming       = "The statement is being resumed"
	scenarioStateStatementHasBeenDeleted   = "The statement has been deleted"
	scenarioStateStatementHasBeenStopped   = "The statement has been stopped"
	scenarioStateStatementHasBeenResumed   = "The statement has been resumed"
	statementScenarioName                  = "confluent_flink_statement Resource Lifecycle"
	statementWithInitialOffsetScenarioName = "confluent_flink_statement (initial offset) Resource Lifecycle"

	flinkPrincipalIdTest                = "u-yo9j87"
	flinkPrincipalUpdatedIdTest         = "sa-yo9j87"
	flinkComputePoolIdTest              = "lfcp-x7rgx1"
	flinkComputePoolUpdatedIdTest       = "lfcp-x7rgx2"
	flinkStatementTest                  = "SELECT CURRENT_TIMESTAMP;"
	flinkStatementWithInitialOffsetTest = "INSERT INTO customers_sink123 (customer_id, name, address, postcode, city, email) SELECT customer_id, name, address, postcode, city, email FROM customers_source123"
	flinkStatementNameTest              = "workspace-2023-11-15-030109-0408d52d-eaff-4d50-a246-f822a29f2eb9"
	flinkFirstPropertyKeyTest           = "sql.local-time-zone"
	flinkFirstPropertyValueTest         = "GMT-08:00"
	flinkSecondPropertyKeyTest          = "sql.current-catalog"
	flinkSecondPropertyValueTest        = "test"
	flinkThirdPropertyKeyTest           = "sql.current-database"
	flinkThirdPropertyValueTest         = "cluster_0"
	flinkStatementResourceLabel         = "example"

	latestOffsetsTimestampEmptyValueTest   = "0001-01-01T00:00:00Z"
	latestOffsetsTimestampStoppedValueTest = "2024-10-14T21:26:07Z"

	latestOffsetFirstValueTest             = "partition:0,offset:9223372036854775808;partition:4,offset:9223372036854775808;partition:3,offset:9223372036854775808;partition:2,offset:9223372036854775808;partition:1,offset:9223372036854775808;partition:5,offset:9223372036854775808"
	latestOffsetFirstKeyTest               = "customers_source"
	flinkCarryOverOffsetsPropertyValueTest = "tf-2025-08-20-171534-5f6e0644-dc74-4c30-81c1-cd9eaede302f"
)

var fullFlinkStatementResourceLabel = fmt.Sprintf("confluent_flink_statement.%s", flinkStatementResourceLabel)
var createFlinkStatementPath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/statements", flinkOrganizationIdTest, flinkEnvironmentIdTest)
var readFlinkStatementPath = fmt.Sprintf("/sql/v1/organizations/%s/environments/%s/statements/%s", flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkStatementNameTest)

func TestAccFlinkStatementWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockFlinkStatementTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockFlinkStatementTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/create_flink_statement.json")
	createFlinkStatementStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateStatementIsPending).
		WillReturn(
			string(createFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkStatementStub)

	readPendingFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_pending_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsPending).
		WillSetStateTo(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readPendingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_running_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readCreatedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Update the Flink statement stopped status false -> true to trigger a stop
	stopFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	stopFlinkStatementStub := wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillSetStateTo(scenarioStateStatementHasBeenStopped).
		WillReturn(
			string(stopFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(stopFlinkStatementStub)

	readStoppedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_stopped_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenStopped).
		WillReturn(
			string(readStoppedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	// Update the Flink statement stopped status true -> false to trigger a resume with different `principal` and `compute_pool`
	resumingFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resuming_flink_statement.json")
	resumingFlinkStatementStub := wiremock.Put(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenStopped).
		WillSetStateTo(scenarioStateStatementIsResuming).
		WillReturn(
			string(resumingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(resumingFlinkStatementStub)

	readResumedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resumed_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsResuming).
		WillSetStateTo(scenarioStateStatementHasBeenResumed).
		WillReturn(
			string(readResumedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readPostResumeFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_resumed_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenResumed).
		WillReturn(
			string(readPostResumeFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenResumed).
		WillSetStateTo(scenarioStateStatementIsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementStub)

	readDeletingFlinkStatementStub := wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsDeleting).
		WillSetStateTo(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(readDeletingFlinkStatementStub)

	readDeletedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement/read_deleted_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			string(readDeletedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkStatementDestroy(s, mockFlinkStatementTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkStatementWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "0"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampEmptyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				Config: testAccCheckFlinkStatementWithEnhancedProviderBlockWithoutComputePool(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "0"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampEmptyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				Config: testAccCheckFlinkStatementStoppedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "true"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("latest_offsets.%s", latestOffsetFirstKeyTest), latestOffsetFirstValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampStoppedValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				Config: testAccCheckFlinkStatementResumedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolUpdatedIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "organization.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "organization.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "environment.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "environment.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("latest_offsets.%s", latestOffsetFirstKeyTest), latestOffsetFirstValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampStoppedValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "1"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkStatementResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fullFlinkStatementResourceLabel].Primary.ID
					statementName, _ := parseStatementName(resourceId)
					return statementName, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createFlinkStatementStub, fmt.Sprintf("POST %s", createFlinkStatementPath), expectedCountOne)
	checkStubCount(t, wiremockClient, stopFlinkStatementStub, fmt.Sprintf("PUT %s", readFlinkStatementPath), expectedCountTwo)
	checkStubCount(t, wiremockClient, deleteFlinkStatementStub, fmt.Sprintf("DELETE %s", readFlinkStatementPath), expectedCountOne)
}

func TestAccFlinkStatementWithInitialOffsetsWithEnhancedProviderBlock(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer wiremockContainer.Terminate(ctx)

	mockFlinkStatementTestServerUrl := wiremockContainer.URI
	confluentCloudBaseUrl := ""
	wiremockClient := wiremock.NewClient(mockFlinkStatementTestServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement_initial_offset/create_flink_statement.json")
	createFlinkStatementStub := wiremock.Post(wiremock.URLPathEqualTo(createFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateStatementIsPending).
		WillReturn(
			string(createFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createFlinkStatementStub)

	readPendingFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement_initial_offset/read_pending_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsPending).
		WillSetStateTo(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readPendingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillReturn(
			string(readPendingFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteFlinkStatementStub := wiremock.Delete(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenCreated).
		WillSetStateTo(scenarioStateStatementIsDeleting).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteFlinkStatementStub)

	readDeletingFlinkStatementStub := wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(scenarioStateStatementIsDeleting).
		WillSetStateTo(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(readDeletingFlinkStatementStub)

	readDeletedFlinkStatementResponse, _ := ioutil.ReadFile("../testdata/flink_statement_initial_offset/read_deleted_flink_statement.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readFlinkStatementPath)).
		InScenario(statementWithInitialOffsetScenarioName).
		WhenScenarioStateIs(scenarioStateStatementHasBeenDeleted).
		WillReturn(
			string(readDeletedFlinkStatementResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			return testAccCheckFlinkStatementDestroy(s, mockFlinkStatementTestServerUrl)
		},
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckFlinkStatementWithInitialOffsetWithEnhancedProviderBlock(confluentCloudBaseUrl, mockFlinkStatementTestServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckFlinkStatementExists(fullFlinkStatementResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "id", fmt.Sprintf("%s/%s/%s", flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkStatementNameTest)),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "compute_pool.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "principal.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "principal.0.id"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement_name", flinkStatementNameTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "statement", flinkStatementWithInitialOffsetTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "stopped", "false"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets.%", "0"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "latest_offsets_timestamp", latestOffsetsTimestampEmptyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "properties.%", "4"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkFirstPropertyKeyTest), flinkFirstPropertyValueTest),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, fmt.Sprintf("properties.%s", flinkCarryOverOffsetsProperty), flinkCarryOverOffsetsPropertyValueTest),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "sql.secrets.openaikey"),
					resource.TestCheckResourceAttr(fullFlinkStatementResourceLabel, "credentials.#", "0"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.key"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "credentials.0.secret"),
					resource.TestCheckNoResourceAttr(fullFlinkStatementResourceLabel, "rest_endpoint"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullFlinkStatementResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					resourceId := resources[fullFlinkStatementResourceLabel].Primary.ID
					statementName, _ := parseStatementName(resourceId)
					return statementName, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createFlinkStatementStub, fmt.Sprintf("POST %s", createFlinkStatementPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteFlinkStatementStub, fmt.Sprintf("DELETE %s", readFlinkStatementPath), expectedCountOne)
}

func testAccCheckFlinkStatementWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      flink_api_key = "%s"
      flink_api_secret = "%s"
      flink_rest_endpoint = "%s"
      flink_principal_id = "%s"
      organization_id = "%s"
      environment_id = "%s"
      flink_compute_pool_id = "%s"
    }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementWithEnhancedProviderBlockWithoutComputePool(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      flink_api_key = "%s"
      flink_api_secret = "%s"
      flink_rest_endpoint = "%s"
      flink_principal_id = "%s"
      organization_id = "%s"
      environment_id = "%s"
    }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementWithInitialOffsetWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
      endpoint = "%s"
      flink_api_key = "%s"
      flink_api_secret = "%s"
      flink_rest_endpoint = "%s"
      flink_principal_id = "%s"
      organization_id = "%s"
      environment_id = "%s"
      flink_compute_pool_id = "%s"
    }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	
	  properties = {
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementWithInitialOffsetTest,
		flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest,
		flinkSecondPropertyKeyTest, flinkSecondPropertyValueTest,
		flinkThirdPropertyKeyTest, flinkThirdPropertyValueTest,
		flinkCarryOverOffsetsProperty, flinkCarryOverOffsetsPropertyValueTest)
}

func testAccCheckFlinkStatementStoppedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
     endpoint = "%s"
     flink_api_key = "%s"
     flink_api_secret = "%s"
     flink_rest_endpoint = "%s"
     flink_principal_id = "%s"
     organization_id = "%s"
     environment_id = "%s"
     flink_compute_pool_id = "%s"
   }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	  stopped = true

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementResumedWithEnhancedProviderBlock(confluentCloudBaseUrl, mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
     endpoint = "%s"
     flink_api_key = "%s"
     flink_api_secret = "%s"
     flink_rest_endpoint = "%s"
     flink_principal_id = "%s"
     organization_id = "%s"
     environment_id = "%s"
     flink_compute_pool_id = "%s"
   }
	resource "confluent_flink_statement" "%s" {
	  statement_name = "%s"
	  statement = "%s"
	  stopped = false

	  properties = {
		"%s" = "%s"
	  }
	}
	`, confluentCloudBaseUrl, kafkaApiKey, kafkaApiSecret, mockServerUrl, flinkPrincipalUpdatedIdTest,
		flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolUpdatedIdTest,
		flinkStatementResourceLabel, flinkStatementNameTest, flinkStatementTest, flinkFirstPropertyKeyTest, flinkFirstPropertyValueTest)
}

func testAccCheckFlinkStatementDestroy(s *terraform.State, url string) error {
	testClient := testAccProvider.Meta().(*Client)
	c := testClient.flinkRestClientFactory.CreateFlinkRestClient(url, flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkComputePoolIdTest, flinkPrincipalIdTest, kafkaApiKey, kafkaApiSecret, false, testClient.oauthToken)
	// Loop through the resources in state, verifying each Kafka topic is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_statement" {
			continue
		}
		deletedTopicId := rs.Primary.ID
		_, response, err := c.apiClient.StatementsSqlV1Api.GetSqlv1Statement(c.apiContext(context.Background()), flinkOrganizationIdTest, flinkEnvironmentIdTest, flinkStatementNameTest).Execute()
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

func testAccCheckFlinkStatementExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s flink statement has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s flink statement", n)
		}

		return nil
	}
}
