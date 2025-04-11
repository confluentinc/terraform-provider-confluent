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

func TestAccDataSourceGcpBYOKKey(t *testing.T) {
	ctx := context.Background()

	wiremockContainer, err := setupWiremock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	mockServerUrl := wiremockContainer.URI
	wiremockClient := wiremock.NewClient(mockServerUrl)

	readGcpKeyResponse, _ := ioutil.ReadFile("../testdata/byok/gcp_key.json")
	readGcpKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(awsKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readGcpKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readGcpKeyStub)

	awsKeyResourceName := "gcp_key"
	fullGcpKeyResourceName := fmt.Sprintf("data.confluent_byok_key.%s", awsKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceGcpByokKeyConfig(mockServerUrl, awsKeyResourceName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "gcp.0.key_id", testGcpByokKeyId),
					resource.TestCheckResourceAttr(fullGcpKeyResourceName, "gcp.0.security_group", testGcpByokSecurityGroup),
				),
			},
		},
	})
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

func testAccCheckDataSourceGcpByokKeyConfig(mockServerUrl, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_byok_key" "%s"{
      id = "cck-abcde"
	}
	`, mockServerUrl, resourceName)
}
