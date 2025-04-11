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

func TestAccDataSourceAwsBYOKKey(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readAwsKeyResponse, _ := ioutil.ReadFile("../testdata/byok/aws_key.json")
	readAwsKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(awsKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAwsKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readAwsKeyStub)

	awsKeyResourceName := "aws_key"
	fullAwsKeyResourceName := fmt.Sprintf("data.confluent_byok_key.%s", awsKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAwsByokKeyConfig(mockServerUrl, awsKeyResourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.key_arn", keyArn),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.0", "arn:aws:iam::111111111111:role/testRoleId1"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.1", "arn:aws:iam::111111111111:role/testRoleId2"),
					resource.TestCheckResourceAttr(fullAwsKeyResourceName, "aws.0.roles.2", "arn:aws:iam::111111111111:role/testRoleId3"),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourceAwsByokKeyConfig(mockServerUrl, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_byok_key" "%s"{
      id = "cck-abcde"
	}
	`, mockServerUrl, resourceName)
}
