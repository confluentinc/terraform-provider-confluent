package provider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/walkerus/go-wiremock"
)

const (
	byokV1Path      = "/byok/v1/keys"
	keyVaultId      = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/test-vault/providers/Microsoft.KeyVault/vaults/test-vault"
	keyUrl          = "https://test-vault.vault.azure.net/keys/test-key/dd554e3117e74ed8bbcd43390e1e3824"
	azureByokTenant = "11111111-1111-1111-1111-111111111111"

	azureKeyScenarioName                = "confluent_azure Key Azure Resource Lifecycle"
	scenarioStateAzureKeyHasBeenDeleted = "The new azure key's deletion has been just completed"
)

func TestAccAzureBYOKKey(t *testing.T) {
	mockServerUrl := tc.wiremockUrl
	wiremockClient := wiremock.NewClient(mockServerUrl)
	// nolint:errcheck
	defer wiremockClient.Reset()

	// nolint:errcheck
	defer wiremockClient.ResetAllScenarios()
	createAzureKeyResponse, _ := ioutil.ReadFile("../testdata/byok/azure_key.json")
	createAzureKeyStub := wiremock.Post(wiremock.URLPathEqualTo(byokV1Path)).
		InScenario(azureKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(createAzureKeyResponse),
			contentTypeJSONHeader,
			http.StatusCreated,
		)

	readAzureKeyResponse, _ := ioutil.ReadFile("../testdata/byok/azure_key.json")
	readAzureKeyStub := wiremock.Get(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(azureKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readAzureKeyResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	deleteAzureKeyStub := wiremock.Delete(wiremock.URLPathEqualTo(fmt.Sprintf("%s/cck-abcde", byokV1Path))).
		InScenario(azureKeyScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillSetStateTo(scenarioStateAzureKeyHasBeenDeleted).
		WillReturn(
			"",
			contentTypeJSONHeader,
			http.StatusNoContent,
		)

	_ = wiremockClient.StubFor(createAzureKeyStub)
	_ = wiremockClient.StubFor(readAzureKeyStub)
	_ = wiremockClient.StubFor(deleteAzureKeyStub)

	azureKeyResourceName := "azure_key"
	fullAzureKeyResourceName := fmt.Sprintf("confluent_byok_key.%s", azureKeyResourceName)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccCheckByokKeyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckAzureByokKeyConfig(mockServerUrl, azureKeyResourceName, azureByokTenant, keyVaultId, keyUrl),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAzureKeyExists(fullAzureKeyResourceName),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "id", "cck-abcde"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.tenant_id", azureByokTenant),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.key_vault_id", "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/test-vault/providers/Microsoft.KeyVault/vaults/test-vault"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.key_identifier", "https://test-vault.vault.azure.net/keys/test-key/dd554e3117e74ed8bbcd43390e1e3824"),
					resource.TestCheckResourceAttr(fullAzureKeyResourceName, "azure.0.application_id", "11111111-1111-1111-1111-111111111111"),
				),
			},
			{
				// https://www.terraform.io/docs/extend/resources/import.html
				ResourceName:      fullAzureKeyResourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
	checkStubCount(t, wiremockClient, createAzureKeyStub, fmt.Sprintf("POST %s", byokV1Path), expectedCountOne)
	checkStubCount(t, wiremockClient, deleteAzureKeyStub, fmt.Sprintf("DELETE %s", fmt.Sprintf("%s/cck-abcde", byokV1Path)), expectedCountOne)
}

func testAccCheckByokKeyDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*Client)
	// Loop through the resources in state, verifying each azure byok key is destroyed
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "confluent_byok_key" {
			continue
		}
		deletedKeyId := rs.Primary.ID
		req := c.byokClient.KeysByokV1Api.GetByokV1Key(c.byokApiContext(tc.ctx), deletedKeyId)
		deletedKey, response, err := req.Execute()
		if response != nil && response.StatusCode == http.StatusNotFound {
			return nil
		} else if err == nil && deletedKey.Id != nil {
			// Otherwise return the error
			if *deletedKey.Id == rs.Primary.ID {
				return fmt.Errorf("key (%s) still exists", rs.Primary.ID)
			}
		}
		return err
	}
	return nil
}

func testAccCheckAzureByokKeyConfig(mockServerUrl, resourceName, tenantId, keyVaultId, keyUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}
	resource "confluent_byok_key" "%s"{
		azure {
		  tenant_id      = "%s"
		  key_vault_id   = "%s"
		  key_identifier = "%s"
	  }
	}
	`, mockServerUrl, resourceName, tenantId, keyVaultId, keyUrl)
}

func testAccCheckAzureKeyExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s Azure Key has not been found", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("ID has not been set for %s Azure Key", resourceName)
		}

		return nil
	}
}
