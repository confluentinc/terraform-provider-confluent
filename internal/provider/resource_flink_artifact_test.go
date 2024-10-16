package provider

import (
	"context"
	"fmt"
	"github.com/walkerus/go-wiremock"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioArtifactPresignedUrlHasBeenCreated = "The new flink artifact predesign URL has been just created"
	scenarioStateFlinkArtifactHasBeenCreated   = "The new flink artifact has been just created"
	scenarioStateFlinkArtifactHasBeenDeleted   = "The new flink artifact has been deleted"
	flinkArtifactScenarioName                  = "confluent_flink_artifact Resource Lifecycle"
	flinkArtifactClass                         = "io.confluent.example.SumScalarFunction"
	flinkArtifactCloud                         = "AWS"
	flinkArtifactRegion                        = "us-east-2"
	flinkArtifactEnvironmentId                 = "env-gz903"
	flinkArtifactContentFormat                 = "JAR"
	flinkArtifactRuntimeLanguage               = "JAVA"
	flinkArtifactDescription                   = "string"
	flinkArtifactId                            = "lfcp-abc123"
	flinkArtifactDisplayName                   = "flink_artifact_0"
	flinkArtifactApiVersion                    = "artifact/v1"
	flinkArtifactKind                          = "FlinkArtifact"
)

var flinkArtifactsUrlPath = fmt.Sprintf("/artifact/v1/flink-artifacts/%s", flinkArtifactId)

func TestAccFlinkArtifact(t *testing.T) {
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

	createArtifactPresignedUrlResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/read_presigned_url.json")
	createArtifactPresignedUrlStub := wiremock.Post(wiremock.URLPathEqualTo("/artifact/v1/presigned-upload-url")).
		InScenario(flinkArtifactScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioArtifactPresignedUrlHasBeenCreated).
		WillReturn(
			string(createArtifactPresignedUrlResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactPresignedUrlStub)

	createArtifactResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/create_artifact.json")
	createArtifactStub := wiremock.Post(wiremock.URLPathEqualTo("/artifact/v1/flink-artifacts")).
		InScenario(flinkArtifactScenarioName).
		WhenScenarioStateIs(scenarioArtifactPresignedUrlHasBeenCreated).
		WillSetStateTo(scenarioStateFlinkArtifactHasBeenCreated).
		WillReturn(
			string(createArtifactResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactStub)

	readCreatedArtifactResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/read_created_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegion)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloud)).
		WhenScenarioStateIs(scenarioStateFlinkArtifactHasBeenCreated).
		WillReturn(
			string(readCreatedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteArtifactStub := wiremock.Delete(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegion)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloud)).
		WhenScenarioStateIs(scenarioStateFlinkArtifactHasBeenCreated).
		WillSetStateTo(scenarioStateFlinkArtifactHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteArtifactStub)

	readDeletedArtifactResponse, _ := ioutil.ReadFile("../testdata/flink_artifact/read_deleted_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegion)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloud)).
		WhenScenarioStateIs(scenarioStateFlinkArtifactHasBeenDeleted).
		WillReturn(
			string(readDeletedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	flinkArtifactResourceLabel := "test"
	fullFlinkArtifactResourceLabel := fmt.Sprintf("confluent_flink_artifact.%s", flinkArtifactResourceLabel)

	_ = os.Setenv("IMPORT_ARTIFACT_FILENAME", "abc.jar")
	defer func() {
		_ = os.Unsetenv("IMPORT_ARTIFACT_FILENAME")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckArtifactDestroy,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckArtifactConfig(mockServerUrl, flinkArtifactResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactExists(fullFlinkArtifactResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramId, flinkArtifactId),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramDisplayName, flinkArtifactDisplayName),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramClass, flinkArtifactClass),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramCloud, flinkArtifactCloud),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramRegion, flinkArtifactRegion),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramArtifactFile, "abc.jar"),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramContentFormat, flinkArtifactContentFormat),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramRuntimeLanguage, flinkArtifactRuntimeLanguage),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramDescription, flinkArtifactDescription),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramApiVersion, flinkArtifactApiVersion),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramKind, flinkArtifactKind),
				),
			},
			{
				ResourceName:      fullFlinkArtifactResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					flinkArtifactId := resources[fullFlinkArtifactResourceLabel].Primary.ID
					region := resources[fullFlinkArtifactResourceLabel].Primary.Attributes["region"]
					cloud := resources[fullFlinkArtifactResourceLabel].Primary.Attributes["cloud"]
					return region + "/" + cloud + "/" + flinkArtifactId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createArtifactStub, fmt.Sprintf("POST %s", flinkArtifactsUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteArtifactStub, fmt.Sprintf("DELETE %s?environment=%s", flinkArtifactsUrlPath, flinkArtifactEnvironmentId), expectedCountOne)
}

func testAccCheckArtifactDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each compute pool is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_artifact" {
			continue
		}
		deletedArtifactId := rs.Primary.ID
		req := c.faClient.FlinkArtifactsArtifactV1Api.GetArtifactV1FlinkArtifact(c.faApiContext(context.Background()), deletedArtifactId).Cloud(flinkArtifactCloud).Region(flinkArtifactRegion)
		deletedArtifact, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedArtifact.Id != nil {
			// Otherwise return the error
			if *deletedArtifact.Id == rs.Primary.ID {
				return fmt.Errorf("artifact (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckArtifactConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	resource "confluent_flink_artifact" "%s" {
		artifact_file    = "abc.jar"
        display_name     = "%s"
        cloud            = "%s"
	    region           = "%s"
		class = "%s"
		description = "%s"
		runtime_language = "%s"
	    environment {
		  id = "%s"
	    }
	}
	`, mockServerUrl, resourceLabel, flinkArtifactDisplayName, flinkArtifactCloud, flinkArtifactRegion, flinkArtifactClass, flinkArtifactDescription, flinkArtifactRuntimeLanguage, flinkArtifactEnvironmentId)
}

func testAccCheckArtifactExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]

		if !ok {
			return fmt.Errorf("%s artifact has not been found", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s artifact", n)
		}

		return nil
	}
}
