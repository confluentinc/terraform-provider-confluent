package provider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

func TestAccDataSourceAzureBYOKKey(t *testing.T) {
	mockServerUrl := tc.wiremockUrl
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()

	readAzureKeyResponse, _ := ioutil.ReadFile("../testdata/byok/azure_key.json")
	readAzureKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(azureKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAzureKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readAzureKeyStub)

	azureKeyResourceName := "azure_key"
	fullAzureKeyResourceName := fmt.Sprintf("data.confluent_byok_key.%s", azureKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceAzureByokKeyConfig(mockServerUrl, azureKeyResourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureKeyExists(fullAzureKeyResourceName),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.tenant_id", azureByokTenant),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.key_vault_id", "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/test-vault/providers/Microsoft.KeyVault/vaults/test-vault"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.key_identifier", "https://test-vault.vault.azure.net/keys/test-key/dd554e3117e74ed8bbcd43390e1e3824"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.application_id", "11111111-1111-1111-1111-111111111111"),
				),
			},
		},
	})
}

func testAccCheckDataSourceAzureByokKeyConfig(mockServerUrl, resourceName string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	data "confluent_byok_key" "%s"{
		id = "cck-abcde"
	}
	`, mockServerUrl, resourceName)
}
