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
	testGcpByokKeyId         = "projects/temp-gear-123456/locations/us-central1/keyRings/byok-test/cryptoKeys/byok-test"
	testGcpByokSecurityGroup = "cck-abcde@confluent.io"

	gcpKeyScenarioName                = "confluent_aws Key Gcp Resource Lifecycle"
	scenarioStateGcpKeyHasBeenDeleted = "The new gcp key's deletion has been just completed"
)

func TestAccGcpBYOKKey(t *testing.T) {
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
	createGcpKeyResponse, _ := ioutil.ReadFile("../testdata/byok/gcp_key.json")
	createGcpKeyStub := wiremock.Post(wiremock.URLPathEqualTo(byokV1Path)).
		InScenario(gcpKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createGcpKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)

	readGcpKeyResponse, _ := ioutil.ReadFile("../testdata/byok/gcp_key.json")
	readGcpKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(gcpKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readGcpKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	deleteGcpKeyStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(gcpKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateGcpKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)

	_ = wiremockClient.StubFor(createGcpKeyStub)
	_ = wiremockClient.StubFor(readGcpKeyStub)
	_ = wiremockClient.StubFor(deleteGcpKeyStub)

	awsKeyResourceName := "gcp_key"
	fullGcpKeyResourceName := fmt.Sprintf("confluent_byok_key.%s", awsKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckByokKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckGcpByokKeyConfig(mockServerUrl, awsKeyResourceName, testGcpByokKeyId),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGcpKeyExists(fullGcpKeyResourceName),
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "gcp.0.key_id", testGcpByokKeyId),
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "gcp.0.security_group", testGcpByokSecurityGroup),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullGcpKeyResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
	checkStubCount(t, wiremockClient, createGcpKeyStub, fmt.Sprintf("POST %s", byokV1Path), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteGcpKeyStub, fmt.Sprintf("DELETE %s", fmt.Sprintf("%s/cck-abcde", byokV1Path)), expectedCountOne)

}

func testAccCheckGcpByokKeyConfig(mockServerUrl, resourceName, keyId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_byok_key" "%s" {
	  gcp {
		key_id = "%s"
	  }
	}
	`, mockServerUrl, resourceName, keyId)
}

func testAccCheckGcpKeyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s Gcp Key has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Gcp Key", resourceName)
		}

		return nil
	}
}
