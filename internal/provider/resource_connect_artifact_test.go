package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	scenarioConnectArtifactPresignedUrlHasBeenCreated = "The new connect artifact presigned URL has been just created"
	scenarioStateConnectArtifactHasBeenCreated        = "The new connect artifact has been just created"
	scenarioStateConnectArtifactIsProvisioning        = "The new connect artifact is in provisioning state"
	scenarioStateConnectArtifactHasBeenDeleted        = "The new connect artifact has been deleted"
	connectArtifactScenarioName                       = "confluent_connect_artifact Resource Lifecycle"
	connectArtifactCloud                              = "AWS"
	connectArtifactEnvironmentId                      = "env-gz903"
	connectArtifactContentFormat                      = "JAR"
	connectArtifactContentFormatZip                   = "ZIP"
	connectArtifactDescription                        = "string"
	connectArtifactId                                 = "lccp-abc123"
	connectArtifactUniqueName                         = "connect_artifact_0"
)

var connectArtifactsUrlPath = fmt.Sprintf("/cam/v1/connect-artifacts/%s", connectArtifactId)

func TestAccConnectArtifact(t *testing.T) {
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

	createArtifactPresignedUrlResponse, _ := os.ReadFile("../testdata/connect_artifact/read_presigned_url.json")
	createArtifactPresignedUrlStub := wiremock.Post(wiremock.URLPathEqualTo("/cam/v1/presigned-upload-url")).
		InScenario(connectArtifactScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioConnectArtifactPresignedUrlHasBeenCreated).
		WillReturn(
			string(createArtifactPresignedUrlResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactPresignedUrlStub)

	createArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/create_artifact.json")
	createArtifactStub := wiremock.Post(wiremock.URLPathEqualTo("/cam/v1/connect-artifacts")).
		InScenario(connectArtifactScenarioName).
		WhenScenarioStateIs(scenarioConnectArtifactPresignedUrlHasBeenCreated).
		WillSetStateTo(scenarioStateConnectArtifactIsProvisioning).
		WillReturn(
			string(createArtifactResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactStub)

	// Add a stub for the provisioning state
	provisioningArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/create_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactIsProvisioning).
		WillSetStateTo(scenarioStateConnectArtifactHasBeenCreated).
		WillReturn(
			string(provisioningArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/read_created_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenCreated).
		WillReturn(
			string(readCreatedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteArtifactStub := wiremock.Delete(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenCreated).
		WillSetStateTo(scenarioStateConnectArtifactHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteArtifactStub)

	readDeletedArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/read_deleted_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenDeleted).
		WillReturn(
			string(readDeletedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	connectArtifactResourceLabel := "test"
	fullConnectArtifactResourceLabel := fmt.Sprintf("confluent_connect_artifact.%s", connectArtifactResourceLabel)

	_ = os.Setenv("IMPORT_ARTIFACT_FILENAME", "abc.jar")
	defer func() {
		_ = os.Unsetenv("IMPORT_ARTIFACT_FILENAME")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectArtifactDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectArtifactConfig(mockServerUrl, connectArtifactResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectArtifactExists(fullConnectArtifactResourceLabel),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramId, connectArtifactId),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramDisplayName, connectArtifactUniqueName),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramCloud, connectArtifactCloud),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), connectArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramArtifactFile, "abc.jar"),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramContentFormat, connectArtifactContentFormat),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramDescription, connectArtifactDescription),
				),
			},
			{
				ResourceName:      fullConnectArtifactResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					connectArtifactId := resources[fullConnectArtifactResourceLabel].Primary.ID
					cloud := resources[fullConnectArtifactResourceLabel].Primary.Attributes["cloud"]
					environment := resources[fullConnectArtifactResourceLabel].Primary.Attributes["environment.0.id"]
					return environment + "/" + cloud + "/" + connectArtifactId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createArtifactStub, fmt.Sprintf("POST %s", connectArtifactsUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteArtifactStub, fmt.Sprintf("DELETE %s", connectArtifactsUrlPath), expectedCountOne)
}

func TestAccConnectArtifactZip(t *testing.T) {
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

	createArtifactPresignedUrlResponse, _ := os.ReadFile("../testdata/connect_artifact/read_presigned_url_zip.json")
	createArtifactPresignedUrlStub := wiremock.Post(wiremock.URLPathEqualTo("/cam/v1/presigned-upload-url")).
		InScenario(connectArtifactScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioConnectArtifactPresignedUrlHasBeenCreated).
		WillReturn(
			string(createArtifactPresignedUrlResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactPresignedUrlStub)

	createArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/create_artifact_zip.json")
	createArtifactStub := wiremock.Post(wiremock.URLPathEqualTo("/cam/v1/connect-artifacts")).
		InScenario(connectArtifactScenarioName).
		WhenScenarioStateIs(scenarioConnectArtifactPresignedUrlHasBeenCreated).
		WillSetStateTo(scenarioStateConnectArtifactIsProvisioning).
		WillReturn(
			string(createArtifactResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)
	_ = wiremockClient.StubFor(createArtifactStub)

	// Add a stub for the provisioning state
	provisioningArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/create_artifact_zip.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactIsProvisioning).
		WillSetStateTo(scenarioStateConnectArtifactHasBeenCreated).
		WillReturn(
			string(provisioningArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	readCreatedArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/read_created_artifact_zip.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenCreated).
		WillReturn(
			string(readCreatedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	deleteArtifactStub := wiremock.Delete(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenCreated).
		WillSetStateTo(scenarioStateConnectArtifactHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)
	_ = wiremockClient.StubFor(deleteArtifactStub)

	readDeletedArtifactResponse, _ := os.ReadFile("../testdata/connect_artifact/read_deleted_artifact.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo(connectArtifactsUrlPath)).
		InScenario(connectArtifactScenarioName).
		WithQueryParam("spec.cloud", wiremock.EqualTo(connectArtifactCloud)).
		WithQueryParam("environment", wiremock.EqualTo(connectArtifactEnvironmentId)).
		WhenScenarioStateIs(scenarioStateConnectArtifactHasBeenDeleted).
		WillReturn(
			string(readDeletedArtifactResponse),
			contentTypeJSONHeader,
			http.StatusNotFound,
		))

	connectArtifactResourceLabel := "test_zip"
	fullConnectArtifactResourceLabel := fmt.Sprintf("confluent_connect_artifact.%s", connectArtifactResourceLabel)

	_ = os.Setenv("IMPORT_ARTIFACT_FILENAME", "abc.zip")
	defer func() {
		_ = os.Unsetenv("IMPORT_ARTIFACT_FILENAME")
	}()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckConnectArtifactDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckConnectArtifactZipConfig(mockServerUrl, connectArtifactResourceLabel),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckConnectArtifactExists(fullConnectArtifactResourceLabel),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramId, connectArtifactId),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramDisplayName, connectArtifactUniqueName),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramCloud, connectArtifactCloud),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, fmt.Sprintf("%s.#", paramEnvironment), "1"),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, fmt.Sprintf("%s.0.%s", paramEnvironment, paramId), connectArtifactEnvironmentId),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramArtifactFile, "abc.zip"),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramContentFormat, connectArtifactContentFormatZip),
					resource.TestCheckResourceAttr(fullConnectArtifactResourceLabel, paramDescription, connectArtifactDescription),
				),
			},
			{
				ResourceName:      fullConnectArtifactResourceLabel,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					resources := state.RootModule().Resources
					connectArtifactId := resources[fullConnectArtifactResourceLabel].Primary.ID
					cloud := resources[fullConnectArtifactResourceLabel].Primary.Attributes["cloud"]
					environment := resources[fullConnectArtifactResourceLabel].Primary.Attributes["environment.0.id"]
					return environment + "/" + cloud + "/" + connectArtifactId, nil
				},
			},
		},
	})

	checkStubCount(t, wiremockClient, createArtifactStub, fmt.Sprintf("POST %s", connectArtifactsUrlPath), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteArtifactStub, fmt.Sprintf("DELETE %s", connectArtifactsUrlPath), expectedCountOne)
}

func testAccCheckConnectArtifactDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each connect artifact is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_connect_artifact" {
			continue
		}
		deletedArtifactId := rs.Primary.ID
		req := c.camClient.ConnectArtifactsCamV1Api.GetCamV1ConnectArtifact(c.camApiContext(context.Background()), deletedArtifactId).
			SpecCloud(connectArtifactCloud).
			Environment(connectArtifactEnvironmentId)
		deletedArtifact, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedArtifact.GetId() != "" {
			// Otherwise return the error
			if deletedArtifact.GetId() == rs.Primary.ID {
				return fmt.Errorf("connect artifact (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckConnectArtifactConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_connect_artifact" "%s" {
		display_name = "%s"
		cloud = "%s"
		artifact_file = "abc.jar"
		content_format = "%s"
		description = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, resourceLabel, connectArtifactUniqueName, connectArtifactCloud, connectArtifactContentFormat, connectArtifactDescription, connectArtifactEnvironmentId)
}

func testAccCheckConnectArtifactZipConfig(mockServerUrl, resourceLabel string) string {
	return fmt.Sprintf(`
	provider "confluent" {
		endpoint = "%s"
	}
	resource "confluent_connect_artifact" "%s" {
		display_name = "%s"
		cloud = "%s"
		artifact_file = "abc.zip"
		content_format = "%s"
		description = "%s"
		environment {
			id = "%s"
		}
	}
	`, mockServerUrl, resourceLabel, connectArtifactUniqueName, connectArtifactCloud, connectArtifactContentFormatZip, connectArtifactDescription, connectArtifactEnvironmentId)
}

func testAccCheckConnectArtifactExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("connect artifact resource %s not found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("connect artifact resource %s has no ID set", n)
		}
		return nil
	}
}
