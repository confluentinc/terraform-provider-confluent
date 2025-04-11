package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const (
	CertificateAuthorityDataSourceScenarioName = "confluent_certificate_authority Data Source Lifecycle"
)

func TestAccDataSourceCertificateAuthority(t *testing.T) {
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

	readCertificateAuthorityResponse, _ := os.ReadFile("../testdata/certificate_authority/create_certificate_authority_crl.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/certificate-authorities/op-abc123")).
		InScenario(CertificateAuthorityDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCertificateAuthorityResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	CertificateAuthorityResourceName := "data.confluent_certificate_authority.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceCertificateAuthority(mockServerUrl, "op-abc123"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "id", "op-abc123"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "display_name", "my-ca"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "description", "example-description"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "certificate_chain_filename", "certificate.pem"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "fingerprints.#", "1"),
					resource.TestCheckTypeSetElemAttr(CertificateAuthorityResourceName, "fingerprints.*", "B1BC968BD4f49D622AA89A81F2150152A41D829C"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "expiration_dates.#", "1"),
					resource.TestCheckTypeSetElemAttr(CertificateAuthorityResourceName, "expiration_dates.*", "2017-07-21 17:32:28 +0000 UTC"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "serial_numbers.#", "1"),
					resource.TestCheckTypeSetElemAttr(CertificateAuthorityResourceName, "serial_numbers.*", "219C542DE8f6EC7177FA4EE8C3705797"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "crl_url", "example.url"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "crl_source", "URL"),
					resource.TestCheckResourceAttr(CertificateAuthorityResourceName, "crl_updated_at", "2017-07-21 17:32:28 +0000 UTC"),
				),
			},
		},
	})
	err = wiremockContainer.Terminate(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

func testAccCheckDataSourceCertificateAuthority(mockServerUrl, resourceId string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_certificate_authority" "main" {
      id = "%s"
	}
	`, mockServerUrl, resourceId)
}
