package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/walkerus/go-wiremock"
)

const dnsRecordDataSourceScenarioName = "confluent_dns_record Data Source Lifecycle"

var dnsRecordDataSourceLabel = fmt.Sprintf("data.%s", dnsRecordResourceLabel)

func TestAccDataSourceDnsRecord(t *testing.T) {
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

	readDnsRecordResponse, _ := os.ReadFile("../testdata/network_dns_record/create_dnsrec.json")
	readDnsRecordStub := wiremock.Get(wiremock.URLPathEqualTo(dnsRecordReadUrlPath)).
		InScenario(dnsRecordDataSourceScenarioName).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readDnsRecordResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		)

	_ = wiremockClient.StubFor(readDnsRecordStub)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceDnsRecord(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "id", "dnsrec-abc123"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "display_name", "prod-dnsrec-1"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "domain", "www.example.com"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "environment.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "environment.0.id", "env-abc123"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "gateway.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "gateway.0.id", "gw-abc123"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "private_link_access_point.#", "1"),
					resource.TestCheckResourceAttr(dnsRecordDataSourceLabel, "private_link_access_point.0.id", "ap-abc123"),
				),
			},
		},
	})

}

func testAccCheckDataSourceDnsRecord(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
	  endpoint = "%s"
	}

	data "confluent_dns_record" "main" {
      id = "dnsrec-abc123"
	  environment {
		id = "env-abc123"
	  }
	}
	`, mockServerUrl)
}
