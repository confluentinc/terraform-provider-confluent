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

	time.Sleep(5 * time.Second)
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
