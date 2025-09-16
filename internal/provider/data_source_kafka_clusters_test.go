package provider

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/walkerus/go-wiremock"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	dataSourceKafkaClustersScenarioName = "confluent_kafka_clusters Data Source Lifecycle"
)

var fullKafkaClustersDataSourceLabel = "data.confluent_kafka_clusters.basic-clusters"

func TestAccDataSourceKafkaClusters(t *testing.T) {
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

	readClustersResponse, _ := ioutil.ReadFile("../testdata/kafka/read_kafkas.json")
	_ = wiremockClient.StubFor(wiremock.Get(wiremock.URLPathEqualTo("/cmk/v2/clusters")).
		InScenario(dataSourceKafkaClustersScenarioName).
		WithQueryParam("environment", wiremock.EqualTo(testEnvironmentId)).
		WhenScenarioStateIs(wiremock.ScenarioStateStarted).
		WillReturn(
			string(readClustersResponse),
			contentTypeJSONHeader,
			http.StatusOK,
		))

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckDataSourceKafkaClustersConfig(mockServerUrl),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.#", "2"),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.id", kafkaClusterId),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.display_name", kafkaDisplayName),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.cloud", kafkaCloud),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.region", kafkaRegion),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.availability", kafkaAvailability),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.kind", kafkaKind),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.api_version", kafkaApiVersion),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.bootstrap_endpoint", kafkaBootstrapEndpoint),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.rest_endpoint", kafkaHttpEndpoint),
					resource.TestCheckResourceAttr(fullKafkaClustersDataSourceLabel, "clusters.0.rbac_crn", kafkaRbacCrn),
				),
			},
		},
	})
}

func testAccCheckDataSourceKafkaClustersConfig(mockServerUrl string) string {
	return fmt.Sprintf(`
	provider "confluent" {
 		endpoint = "%s"
	}
	data "confluent_kafka_clusters" "basic-clusters" {
	  	environment {
			id = "%s"
	  	}
	}
	`, mockServerUrl, testEnvironmentId)
}
