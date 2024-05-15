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
	createBusinessMetadataBindingSrUrlPath        = "/catalog/v1/entity/businessmetadata"
	readCreatedBusinessMetadataBindingSrUrlPath   = "/catalog/v1/entity/type/sr_schema/name/lsrc-nrndwv:.:100001/businessmetadata"
	readUpdatedBusinessMetadataBindingSrUrlPath   = "/catalog/v1/entity/type/sr_schema/name/lsrc-nrndwv:.:100002/businessmetadata"
	deleteCreatedBusinessMetadataBindingSrUrlPath = "/catalog/v1/entity/type/sr_schema/name/lsrc-nrndwv:.:100002/businessmetadata/bm"
)

func TestAccBusinessMetadataBindingSrSchema(t *testing.T) {
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

	createBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/create_business_metadata_binding_srschema.json")
	_ = wiremockClient.StubFor(wiremock.Post(wiremock.URLPathEqualTo(createBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenPending).
		WillReturn(
			string(createBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenPending).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_created_business_metadata_binding_srschema.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readCreatedBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillReturn(
			string(readBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	updateBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/update_business_metadata_binding_srschema.json")
	_ = wiremockClient.StubFor(wiremock.Put(wiremock.URLPathEqualTo(createBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenCreated).
		WillSetStateTo(scenarioStateBusinessMetadataBindingHasBeenUpdated).
		WillReturn(
			string(updateBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		))

	readUpdatedBusinessMetadataBindingResponse, _ := ioutil.ReadFile("../testdata/business_metadata/read_updated_business_metadata_binding_srschema.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(readUpdatedBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
		WhenScenarioStateIs(scenarioStateBusinessMetadataBindingHasBeenUpdated).
		WillReturn(
			string(readUpdatedBusinessMetadataBindingResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	_ = wiremockClient.StubFor(wiremock.Delete(wiremock.URLPathEqualTo(deleteCreatedBusinessMetadataBindingSrUrlPath)).
		InScenario(businessMetadataBindingResourceScenarioName).
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
				Config: businessMetadataBindingResourceSchemaConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityName, "lsrc-nrndwv:.:100001"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityType, "sr_schema"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramId, "xxx/bm/lsrc-nrndwv:.:100001/sr_schema"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.%%", paramAttributes), "2"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
				),
			},
			{
				Config: updateBusinessMetadataBindingResourceSchemaConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramBusinessMetadataName, "bm"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityName, "lsrc-nrndwv:.:100002"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramEntityType, "sr_schema"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, paramId, "xxx/bm/lsrc-nrndwv:.:100002/sr_schema"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.%%", paramAttributes), "2"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr1", paramAttributes), "value1"),
					resource.TestCheckResourceAttr(businessMetadataBindingLabel, fmt.Sprintf("%s.attr2", paramAttributes), "value2"),
				),
			},
		},
	})
}

func businessMetadataBindingResourceSchemaConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata_binding" "main" {
	  business_metadata_name = "bm"
	  entity_name = "lsrc-nrndwv:.:100001"
	  entity_type = "sr_schema"
	  attributes = {
		"attr1" = "value1"
		"attr2" = "value2"
	  }
	}
 	`, mockServerUrl)
}

func updateBusinessMetadataBindingResourceSchemaConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
 	provider "confluent" {
 	  schema_registry_id = "xxx"
	  schema_registry_rest_endpoint = "%s" # optionally use SCHEMA_REGISTRY_REST_ENDPOINT env var
	  schema_registry_api_key       = "x"       # optionally use SCHEMA_REGISTRY_API_KEY env var
	  schema_registry_api_secret = "x"
 	}
 	resource "confluent_business_metadata_binding" "main" {
	  business_metadata_name = "bm"
	  entity_name = "lsrc-nrndwv:.:100002"
	  entity_type = "sr_schema"
	  attributes = {
		"attr1" = "value1"
		"attr2" = "value2"
	  }
	}
 	`, mockServerUrl)
}
