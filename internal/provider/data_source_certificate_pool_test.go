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
	CertificatePoolDataSourceScenarioName = "confluent_certificate_pool Data Source Lifecycle"
)

func TestAccDataSourceCertificatePool(t *testing.T) {
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

	readCertificatePoolResponse, _ := os.ReadFile("../testdata/certificate_pool/create_certificate_pool.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/certificate-authorities/op-abc123/identity-pools/pool-def456")).
		InScenario(CertificatePoolDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readCertificatePoolResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	listCertificatePoolsResponse, _ := os.ReadFile("../testdata/certificate_pool/list_certificate_pools.json")
	_ = wiremockClient.StubFor(
		wiremock.Get(wiremock.URLPathEqualTo("/iam/v2/certificate-authorities/op-abc123/identity-pools")).
			WithQueryParam("page_size", wiremock.EqualTo("99")).
			InScenario(CertificatePoolDataSourceScenarioName).
			WhenScenarioStateIs(wiremock.ScenarioStateStarted).
			WillReturn(
				string(listCertificatePoolsResponse),
				contentTypeJSONHeader,
				http.StatusOK,
			),
	)

	CertificatePoolResourceName := "data.confluent_certificate_pool.main"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceCertificatePool(mockServerUrl, "pool-def456"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "id", certificatePoolId),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "display_name", "my-certificate-pool"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "description", "example-description"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "external_identifier", "UID"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "filter", "C=='Canada' && O=='Confluent'"),
				),
			},
			{
				Config: testAccCheckDataSourceCertificatePoolDisplayName(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "id", certificatePoolId),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "display_name", "my-certificate-pool"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "description", "example-description"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "external_identifier", "UID"),
					resource.TestCheckResourceAttr(CertificatePoolResourceName, "filter", "C=='Canada' && O=='Confluent'"),
				),
			},
		},
	})

}

func testAccCheckDataSourceCertificatePool(mockServerUrl, resourceId string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	data "confluent_certificate_pool" "main" {
	    id = "%s"
		certificate_authority {
		    id = "op-abc123"
		}
	}
	`, mockServerUrl, resourceId)
}

func testAccCheckDataSourceCertificatePoolDisplayName(mockServerUrl string) string {
	return fmt.Sprintf(`
    provider "confluent" {
        endpoint = "%s"
    }

	data "confluent_certificate_pool" "main" {
	    display_name = "my-certificate-pool"
		certificate_authority {
		    id = "op-abc123"
		}
	}
	`, mockServerUrl)
}
