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
	dataSourceFlinkArtifactScenarioName = "confluent_flink_artifact Data Source Lifecycle"
)

var fullArtifactDataSourceLabel = fmt.Sprintf("data.confluent_flink_artifact.%s", networkDataSourceLabel)

func TestAccDataSourceFlinkArtifact(t *testing.T) {
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

	readCreatedFlinkArtifactResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/read_created_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/artifact/v1/flink-artifacts/lfcp-abc123")).
		InScenario(dataSourceFlinkArtifactScenarioName).
		WithQueryParam("cloud", wiremock.EqualTo("AWS")).
		WithQueryParam("region", wiremock.EqualTo("us-east-2")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCreatedFlinkArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readArtifactsResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/read_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/artifact/v1/flink-artifacts")).
		InScenario(dataSourceFlinkArtifactScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(flinkArtifactEnvironmentId)).
		WithQueryParam("cloud", wiremock.EqualTo("AWS")).
		WithQueryParam("region", wiremock.EqualTo("us-east-2")).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readArtifactsResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceFlinkArtifactConfigWithIdSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactExists(fullArtifactDataSourceLabel),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramId, flinkArtifactId),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramDisplayName, flinkArtifactDisplayName),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramClass, flinkArtifactClass),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramCloud, flinkArtifactCloud),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramRegion, flinkArtifactRegion),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramContentFormat, flinkArtifactContentFormat),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramRuntimeLanguage, flinkArtifactRuntimeLanguage),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramDescription, flinkArtifactDescription),
				),
			},
			{
				Config: testAccCheckDataSourceFlinkArtifactConfigWithDisplayNameSet(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactExists(fullArtifactDataSourceLabel),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramId, flinkArtifactId),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramDisplayName, flinkArtifactDisplayName),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramClass, flinkArtifactClass),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramCloud, flinkArtifactCloud),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramRegion, flinkArtifactRegion),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramContentFormat, flinkArtifactContentFormat),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramRuntimeLanguage, flinkArtifactRuntimeLanguage),
					resource.TestCheckResourceAttr(fullArtifactDataSourceLabel, paramDescription, flinkArtifactDescription),
				),
			},
		},
	})
}

func testAccCheckDataSourceFlinkArtifactConfigWithDisplayNameSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_flink_artifact" "%s" {
		display_name = "%s"
		cloud = "%s"
		region = "%s"
		class = "%s"
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, networkDataSourceLabel, flinkArtifactDisplayName, flinkArtifactCloud, flinkArtifactRegion, flinkArtifactClass,
		flinkArtifactEnvironmentId)
}

func testAccCheckDataSourceFlinkArtifactConfigWithIdSet(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_flink_artifact" "%s" {
	    id = "%s"
		cloud = "%s"
		region = "%s"
		class = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, networkDataSourceLabel, flinkArtifactId, flinkArtifactCloud, flinkArtifactRegion, flinkArtifactClass, flinkArtifactEnvironmentId)
}
