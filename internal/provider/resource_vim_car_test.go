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
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/walkerus/go-wiremock"
)

const (
	scenarioStateCarHasBeenCreated             = "The new car has been just created"
	scenarioStateCarManufacturerHasBeenUpdated = "The car's manufacturer has been just updated"
	scenarioStateCarHasBeenDeleted             = "The car has been deleted"
	carScenarioName                            = "confluent_vim_car Resource Lifecycle"
	carId                                      = "car-abc123"
)

func TestAccVimCar(t *testing.T) {
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

	createCarResponse, _ := ioutil.ReadFile("../testdata/vim_car/create_car.json")
	createCarStub := wiremock.Post(wiremock.URLPathEqualTo("/vim/v1/cars")).
		InScenario(carScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateCarHasBeenCreated).
		WillReturn(
			string(createCarResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createCarStub)

	readCreatedCarResponse, _ := ioutil.ReadFile("../testdata/vim_car/read_created_car.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/vim/v1/cars/%s", carId))).
		InScenario(carScenarioName).
		WhenScenarioStateIs(scenarioStateCarHasBeenCreated).
		WillReturn(
			string(readCreatedCarResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readUpdatedCarResponse, _ := ioutil.ReadFile("../testdata/vim_car/read_updated_car.json")
	patchCarStub := wiremock.Patch(wiremock.URLPathEqualTo(fmt.Sprintf("/vim/v1/cars/%s", carId))).
		InScenario(carScenarioName).
		WhenScenarioStateIs(scenarioStateCarHasBeenCreated).
		WillSetStateTo(scenarioStateCarManufacturerHasBeenUpdated).
		WillReturn(
			string(readUpdatedCarResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)
	_ = wiremockClient.StubFor(patchCarStub)

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/vim/v1/cars/%s", carId))).
		InScenario(carScenarioName).
		WhenScenarioStateIs(scenarioStateCarManufacturerHasBeenUpdated).
		WillReturn(
			string(readUpdatedCarResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readDeletedCarResponse, _ := ioutil.ReadFile("../testdata/vim_car/read_deleted_car.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("/vim/v1/cars/%s", carId))).
		InScenario(carScenarioName).
		WhenScenarioStateIs(scenarioStateCarHasBeenDeleted).
		WillReturn(
			string(readDeletedCarResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	deleteCarStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("/vim/v1/cars/%s", carId))).
		InScenario(carScenarioName).
		WhenScenarioStateIs(scenarioStateCarManufacturerHasBeenUpdated).
		WillSetStateTo(scenarioStateCarHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteCarStub)

	manufacturer := "Toyota"
	model := "Camry"
	// in order to test tf update (step #3)
	updatedManufacturer := "Honda"
	carResourceLabel := "test_car_resource_label"
	fullCarResourceLabel := fmt.Sprintf("confluent_vim_car.%s", carResourceLabel)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckVimCarDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckVimCarConfig(mockServerUrl, carResourceLabel, manufacturer, model),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVimCarExists(fullCarResourceLabel),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "id", carId),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "manufacturer", manufacturer),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "model", model),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "webhook_endpoint", "https://webhook.example.com/cars/car-abc123"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullCarResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccCheckVimCarConfig(mockServerUrl, carResourceLabel, updatedManufacturer, model),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVimCarExists(fullCarResourceLabel),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "id", carId),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "manufacturer", updatedManufacturer),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "model", model),
					resource.TestCheckResourceAttr(fullCarResourceLabel, "webhook_endpoint", "https://webhook.example.com/cars/car-abc123"),
				),
			},
			{
				ResourceName:      fullCarResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})

	checkStubCount(t, wiremockClient, createCarStub, "POST /vim/v1/cars", expectedCountOne)
	checkStubCount(t, wiremockClient, patchCarStub, fmt.Sprintf("PATCH /vim/v1/cars/%s", carId), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteCarStub, fmt.Sprintf("DELETE /vim/v1/cars/%s", carId), expectedCountOne)
}

func testAccCheckVimCarDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each car is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_vim_car" {
			continue
		}
		deletedCarId := rs.Primary.ID
		req := c.vimClient.CarsVimV1Api.GetVimV1Car(c.vimApiContext(context.Background()), deletedCarId)
		deletedCar, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			// If the error is equivalent to http.StatusNotFound, the car is destroyed.
			return nil
		} else if err == nil && deletedCar.Id != nil {
			// Otherwise return the error
			if *deletedCar.Id == rs.Primary.ID {
				return fmt.Errorf("car (%q) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckVimCarConfig(mockServerUrl, carResourceLabel, manufacturer, model string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }
    resource "confluent_vim_car" "%s" {
        manufacturer = "%s"
        model        = "%s"
    }
    `, mockServerUrl, carResourceLabel, manufacturer, model)
}

func testAccCheckVimCarExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s car has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s car", n)
		}

		return nil
	}
}
