//go:build pact || pact.consumer

package provider

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pact-foundation/pact-go/v2/consumer"
	pactlog "github.com/pact-foundation/pact-go/v2/log"
	m "github.com/pact-foundation/pact-go/v2/matchers"
	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/stretchr/testify/require"

	cmk "github.com/confluentinc/ccloud-sdk-go-v2/cmk/v2"
)

func TestConsumer(t *testing.T) {
	pactlog.SetLogLevel("DEBUG")
	mockProvider, err := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
		Consumer: "terraform-provider-confluent",
		Provider: "TODO: confluent API service name",
		// TLS: true,
	})
	require.NoError(t, err)

	kafkaClusterId := "lkc-12345"
	environmentId := "env-abc123"
	region := "us-east4"

	err = mockProvider.AddInteraction().
		GivenWithParameter(models.ProviderState{
			Name: "kafka cluster exists",
			Parameters: map[string]interface{}{
				"cluster_id": kafkaClusterId,
			},
		}).
		UponReceiving("A request to get kafka cluster by id").
		WithRequest("GET", fmt.Sprintf("/cmk/v2/clusters/%v", kafkaClusterId), func(b *consumer.V4RequestBuilder) {
			b.
				// Header("Content-Type", m.Equality("application/json")).
				// Header("Accepts", m.Equality("application/json")).
				// Header("Authorization", m.S("TODO: token")).
				Query("environment", m.S(environmentId))
			// BodyMatch(m.S(""))
		}).
		// TODO: ideally a single consumer should only care about fields it actually consumes.
		// So the contract != spec. Trim down this contract to only the fields we actually use.
		// seems like the usage point is setKafkaClusterAttributes in resource_kafka_cluster.go
		// called by data_source_kafka_cluster.go
		// TODO: don't use regexes for cases where consumer doesn't care about a specific format - e.g., consumer
		// probably doesn't care that `resource_name` starts with `crn://`?
		// Only make assertions about things that will affect the consumer if they change:
		// https://docs.pact.io/consumer#only-make-assertions-about-things-that-will-affect-the-consumer-if-they-change
		WillRespondWith(200, func(b *consumer.V4ResponseBuilder) {
			b.
				Header("Content-Type", m.Equality("application/json")).
				JSONBody(m.StructMatcher{
					// Does the client actually care what the API version is?
					"api_version": m.Equality("cmk/v2"),
					"kind":        m.Equality("Cluster"),
					"id":          m.Equality(kafkaClusterId),
					"metadata": m.Map{
						"self": m.Regex(fmt.Sprintf("http://127.0.0.1/cmk/v2/clusters/%v", kafkaClusterId), `^(http|https)://.+`),
						// utils.go::clusterCrnToRbacClusterCrn cares about /kafka=xxx suffix
						"resource_name": m.Regex(
							fmt.Sprintf("crn://confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=%v/cloud-cluster=%v", environmentId, kafkaClusterId),
							`^crn://.+/kafka=.+`,
						),
						// TODO: ensure timestamp is in correct format - web example has "2006-01-02T15:04:05-07:00"
						// both m.Timestamp() and our web example claim RFC3339 format,
						// but web has -07:00 and m.Timestamp() has Z07:00
						"created_at": m.Timestamp(),
						"updated_at": m.Timestamp(),
						"deleted_at": m.Timestamp(),
					},
					"spec": m.StructMatcher{
						"display_name": m.S("ProdKafkaCluster"),
						"availability": m.Regex("SINGLE_ZONE", "^(SINGLE_ZONE|MULTI_ZONE)$"),
						"cloud":        m.Regex("GCP", "^(AWS|GCP|AZURE)$"),
						"region":       m.S(region),
						"config": m.Map{
							// TODO: some cluster types require additional fields; add separate tests for those
							"kind": m.Regex("Basic", "^(Basic|Standard|Dedicated|Enterprise)$"),
						},
						// TODO: smart matching here
						"kafka_bootstrap_endpoint": m.S(fmt.Sprintf("%v.%v.gcp.glb.confluent.cloud:9092", kafkaClusterId, region)),
						"http_endpoint":            m.Regex("https://lkc-00000-00000.us-central1.gcp.glb.confluent.cloud", `^(http|https)://.+`),
						"api_endpoint":             m.Regex("https://pkac-00000.us-west-2.aws.confluent.cloud", `^(http|https)://.+`),
						"environment": m.Map{
							"id":            m.S("env-00000"),
							"environment":   m.S("string"),
							"related":       m.Regex("https://api.confluent.cloud/v2/environments/env-00000", `^(http|https)://.+`),
							"resource_name": m.Regex("https://api.confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-00000", `^(http|https)://.+`),
						},
						"network": m.Map{
							"id":            m.S("n-00000"),
							"environment":   m.S("string"),
							"related":       m.Regex("https://api.confluent.cloud/networking/v1/networks/n-00000", `^(http|https)://.+`),
							"resource_name": m.Regex("https://api.confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/environment=env-abc123/network=n-00000", `^(http|https)://.+`),
						},
						"byok": m.Map{
							"id":            m.S("cck-00000"),
							"related":       m.Regex("https://api.confluent.cloud/byok/v1/keys/cck-00000", `^(http|https)://.+`),
							"resource_name": m.Regex("https://api.confluent.cloud/organization=9bb441c4-edef-46ac-8a41-c49e44a3fd9a/key=cck-00000", `^(http|https)://.+`),
						},
					},
					"status": m.Map{
						"phase": m.Regex("PROVISIONED", "^(PROVISIONED|PROVISIONING|FAILED)$"),
						"cku":   m.Integer(2),
					},
				})
		}).
		ExecuteTest(t, func(config consumer.MockServerConfig) error {
			// config.TLSConfig.InsecureSkipVerify = true
			// httpClient := &http.Client{
			// 	Transport: &http.Transport{
			// 		TLSClientConfig: config.TLSConfig,
			// 	},
			// }
			cmkCfg := cmk.NewConfiguration()
			// cmkCfg.HTTPClient = httpClient
			cmkCfg.HTTPClient = &http.Client{}
			cmkCfg.Servers[0].URL = fmt.Sprintf("http://%v:%v", config.Host, config.Port)

			data := &schema.ResourceData{}
			client := Client{
				cmkClient: cmk.NewAPIClient(cmkCfg),
			}
			result, err := readKafkaClusterAndSetAttributes(
				context.Background(), data, &client, environmentId, kafkaClusterId)
			require.NoError(t, err)
			require.Equal(t, "ProdKafkaCluster", result[0].Get(paramDisplayName))
			return nil
		})
	require.NoError(t, err)
}
