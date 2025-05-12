package provider

import (
	"context"
	"fmt"
	"github.com/walkerus/go-wiremock"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	flinkArtifactCloudAzure  = "AZURE"
	flinkArtifactRegionAzure = "centralus"
)

func TestAccFlinkArtifactAzure(t *testing.T) {
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

	createArtifactPresignedUrlResponse, _ := os.ReadFile("../testdata/flink_artifact/read_presigned_url.json")
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

	createArtifactResponse, _ := os.ReadFile("../testdata/flink_artifact/create_artifact_azure.json")
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

	readCreatedArtifactResponse, _ := os.ReadFile("../testdata/flink_artifact/read_created_artifact_azure.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegionAzure)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloudAzure)).
		WithQueryParam("environment", wiremock.EqualTo(flinkArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateFlinkArtifactHasBeenCreated).
		WillReturn(
			string(readCreatedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteArtifactStub := wiremock.Delete(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegionAzure)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloudAzure)).
		WithQueryParam("environment", wiremock.EqualTo(flinkArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateFlinkArtifactHasBeenCreated).
		WillSetStateTo(scenarioStateFlinkArtifactHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteArtifactStub)

	readDeletedArtifactResponse, _ := os.ReadFile("../testdata/flink_artifact/read_deleted_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(flinkArtifactsUrlPath)).
		InScenario(flinkArtifactScenarioName).
		WithQueryParam("region", wiremock.EqualTo(flinkArtifactRegionAzure)).
		WithQueryParam("cloud", wiremock.EqualTo(flinkArtifactCloudAzure)).
		WithQueryParam("environment", wiremock.EqualTo(flinkArtifactEnvironmentId)).
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
		CheckDestroy:      testAccCheckArtifactDestroyAzure,
		// https://www.terraform.io/docs/extend/testing/acceptance-tests/teststep.html
		// https://www.terraform.io/docs/extend/best-practices/testing.html#built-in-patterns
		Steps: []resource.TestStep{
			{
				Config: testAccCheckArtifactConfig(mockServerUrl, flinkArtifactResourceLabel, flinkArtifactCloudAzure, flinkArtifactRegionAzure),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckArtifactExists(fullFlinkArtifactResourceLabel),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramId, flinkArtifactId),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramDisplayName, flinkArtifactUniqueName),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramClass, flinkArtifactClass),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramCloud, flinkArtifactCloudAzure),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramRegion, flinkArtifactRegionAzure),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), flinkArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramArtifactFile, "abc.jar"),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramContentFormat, flinkArtifactContentFormat),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramRuntimeLanguage, flinkArtifactRuntimeLanguage),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramDescription, flinkArtifactDescription),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramDocumentationLink, flinkArtifactDocumentationLink),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramApiVersion, flinkArtifactApiVersion),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, paramKind, flinkArtifactKind),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, "versions.#", "1"),
					resource.TestCheckResourceAttr(fullFlinkArtifactResourceLabel, "versions.0.version", flinkVersions),
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
					environment := resources[fullFlinkArtifactResourceLabel].Primary.Attributes["environment.0.id"]
					return environment + "/" + region + "/" + cloud + "/" + flinkArtifactId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createArtifactStub, fmt.Sprintf("POST %s", flinkArtifactsUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteArtifactStub, fmt.Sprintf("DELETE %s?environment=%s", flinkArtifactsUrlPath, flinkArtifactEnvironmentId), expectedCountOne)
}

func testAccCheckArtifactDestroyAzure(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each compute pool is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_flink_artifact" {
			continue
		}
		deletedArtifactId := rs.Primary.ID
		req := c.faClient.FlinkArtifactsArtifactV1Api.GetArtifactV1FlinkArtifact(c.faApiContext(context.Background()), deletedArtifactId).Cloud(flinkArtifactCloudAzure).Region(flinkArtifactRegionAzure).Environment(flinkArtifactEnvironmentId)
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
