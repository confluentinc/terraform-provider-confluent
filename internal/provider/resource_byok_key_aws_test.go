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
	keyArn = "arn:aws:kms:us-west-2:111111111111:key/11111111-1111-1111-1111-111111111111"

	awsKeyScenarioName                = "confluent_aws Key Aws Resource Lifecycle"
	scenarioStateAwsKeyHasBeenDeleted = "The new aws key's deletion has been just completed"
)

func TestAccAwsBYOKKey(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	createAwsKeyResponse, _ := ioutil.ReadFile("../testdata/byok/aws_key.json")
	createAwsKeyStub := wiremock.Post(wiremock.URLPathEqualTo(byokV1Path)).
		InScenario(awsKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createAwsKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)

	readAwsKeyResponse, _ := ioutil.ReadFile("../testdata/byok/aws_key.json")
	readAwsKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(awsKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	deleteAwsKeyStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(awsKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAwsKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)

	_ = wiremockClient.StubFor(createAwsKeyStub)
	_ = wiremockClient.StubFor(readAwsKeyStub)
	_ = wiremockClient.StubFor(deleteAwsKeyStub)

	awsKeyResourceName := "aws_key"
	fullAwsKeyResourceName := fmt.Sprintf("confluent_byok_key.%s", awsKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckByokKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAwsByokKeyConfig(mockServerUrl, awsKeyResourceName, keyArn),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAwsKeyExists(fullAwsKeyResourceName),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.key_arn", keyArn),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.0", "arn:aws:iam::111111111111:role/testRoleId1"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.1", "arn:aws:iam::111111111111:role/testRoleId2"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.2", "arn:aws:iam::111111111111:role/testRoleId3"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullAwsKeyResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
	checkStubCount(t, wiremockClient, createAwsKeyStub, fmt.Sprintf("POST %s", byokV1Path), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAwsKeyStub, fmt.Sprintf("DELETE %s", fmt.Sprintf("%s/cck-abcde", byokV1Path)), expectedCountOne)

	t.Cleanup(func() {
		err := wiremockClient.Reset()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset wiremock: %v", err))
		}

		err = wiremockClient.ResetAllScenarios()
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to reset scenarios: %v", err))
		}

		// Also add container termination here to ensure it happens
		err = wiremockContainer.Terminate(ctx)
		if err != nil {
			t.Fatal(fmt.Sprintf("Failed to terminate container: %v", err))
		}
	})

}

func testAccCheckAwsByokKeyConfig(mockServerUrl, resourceName, keyArn string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_byok_key" "%s" {
	  aws {
		key_arn = "%s"
	  }
	}
	`, mockServerUrl, resourceName, keyArn)
}

func testAccCheckAwsKeyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s Aws Key has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Aws Key", resourceName)
		}

		return nil
	}
}
