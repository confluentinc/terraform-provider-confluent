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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

func TestAccKek(t *testing.T) {
	ctx := context.Background()

	initialContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer initialContainer.Terminate(ctx)

	updatedContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer updatedContainer.Terminate(ctx)

	mockServerInitialUrl := initialContainer.URI
	mockServerUpdatedUrl := updatedContainer.URI
	initialClient := wiremock.NewClient(mockServerInitialUrl)
	updatedClient := wiremock.NewClient(mockServerUpdatedUrl)
	// nolint:errcheck
	defer initialClient.Reset()
	defer updatedClient.Reset()

	// nolint:errcheck
	defer initialClient.ResetAllScenarios()
	defer updatedClient.ResetAllScenarios()

	// WireMock scenario state does not transfer between containers. When step 2 switches
	// to mockServerUpdatedUrl, advance updatedClient from Started to KekHasBeenCreated so
	// it can serve reads/updates without the resource being recreated.
	dummyPath := "/state-sync"
	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(dummyPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKekHasBeenCreated).
		WillReturn("OK", contentTypeJSONHeader, http.StatusOK))
	http.Get(mockServerUpdatedUrl + dummyPath)

	createKekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_kek/kek.json")
	createKekStub := wiremock.Post(wiremock.URLPathEqualTo(createKekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateKekHasBeenCreated).
		WillReturn(
			string(createKekResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = initialClient.StubFor(createKekStub)

	_ = initialClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenCreated).
		WillReturn(
			string(createKekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenCreated).
		WillReturn(
			string(createKekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateKekResponse, _ := ioutil.ReadFile("../testdata/schema_registry_kek/updated_kek.json")
	_ = updatedClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenCreated).
		WillSetStateTo(scenarioStateKekHasBeenUpdated).
		WillReturn(
			string(updateKekResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = updatedClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WhenScenarioStateIs(scenarioStateKekHasBeenUpdated).
		WillReturn(
			string(updateKekResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteKekStub := wiremock.Delete(wiremock.URLPathEqualTo(kekUrlPath)).
		InScenario(kekResourceScenarioName).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = updatedClient.StubFor(deleteKekStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: kekResourceConfig(mockServerInitialUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kekLabel, "id", "111/testkek"),
					resource.TestCheckResourceAttr(kekLabel, "name", "testkek"),
					resource.TestCheckResourceAttr(kekLabel, "kms_type", "aws-kms"),
					resource.TestCheckResourceAttr(kekLabel, "kms_key_id", "kmsKeyId"),
					resource.TestCheckResourceAttr(kekLabel, "rest_endpoint", mockServerInitialUrl),
					resource.TestCheckResourceAttr(kekLabel, "shared", "false"),
					resource.TestCheckResourceAttr(kekLabel, "hard_delete", "false"),
					resource.TestCheckResourceAttr(kekLabel, "doc", ""),
				),
			},
			{
				Config: kekResourceUpdatedConfig(mockServerUpdatedUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(kekLabel, "id", "111/testkek"),
					resource.TestCheckResourceAttr(kekLabel, "name", "testkek"),
					resource.TestCheckResourceAttr(kekLabel, "kms_type", "aws-kms"),
					resource.TestCheckResourceAttr(kekLabel, "kms_key_id", "kmsKeyId"),
					resource.TestCheckResourceAttr(kekLabel, "rest_endpoint", mockServerUpdatedUrl),
					resource.TestCheckResourceAttr(kekLabel, "shared", "false"),
					resource.TestCheckResourceAttr(kekLabel, "hard_delete", "false"),
					resource.TestCheckResourceAttr(kekLabel, "doc", "new description"),
				),
			},
		},
	})

	checkStubCount(t, initialClient, createKekStub, fmt.Sprintf("POST %s", createKekUrlPath), expectedCountOne)
	checkStubCount(t, updatedClient, deleteKekStub, fmt.Sprintf("DELETE %s", kekUrlPath), expectedCountOne)
}

func kekResourceConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	}
 	resource "confluent_schema_registry_kek" "mykek" {
	  schema_registry_cluster {
	    id = "111"
	  }
	  rest_endpoint = "%s"
	  credentials {
	    key    = "x"
	    secret = "x"
	  }
	  name = "testkek"
	  kms_type = "aws-kms"
	  kms_key_id = "kmsKeyId"
	  shared = false
	}

 	`, mockServerUrl)
}

func kekResourceUpdatedConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	}
 	resource "confluent_schema_registry_kek" "mykek" {
	  schema_registry_cluster {
	    id = "111"
	  }
	  rest_endpoint = "%s"
	  credentials {
	    key    = "x"
	    secret = "x"
	  }
	  name = "testkek"
	  kms_type = "aws-kms"
	  kms_key_id = "kmsKeyId"
	  doc = "new description"
	  shared = false
	}
 	`, mockServerUrl)
}
