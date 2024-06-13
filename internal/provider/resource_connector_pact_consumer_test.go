//go:build pact || pact.consumer

package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/pact-foundation/pact-go/v2/consumer"
	m "github.com/pact-foundation/pact-go/v2/matchers"
	"github.com/pact-foundation/pact-go/v2/models"
	"github.com/stretchr/testify/require"

	connect "github.com/confluentinc/ccloud-sdk-go-v2/connect/v1"
)

func TestReadConnector(t *testing.T) {
	mockProvider, err := consumer.NewV4Pact(consumer.MockHTTPProviderConfig{
		Consumer: "terraform-provider-confluent",
		Provider: "cc-control-plane-connect",
		PactDir:  os.Getenv("PACT_FILES_DIR"),
		// TODO: figure out how to make TLS work, needs certificate:
		// failed to verify certificate: x509: “localhost” certificate is not standards compliant
		// TLS: true,
	})
	require.NoError(t, err)

	connectorName := "test-connector"
	environmentId := "the-account" // hardcoded in the provider and SRG, consider allowing parametrization
	kafkaClusterId := "cluster-123"
	logicalClusterId := "lcc-123"
	url := fmt.Sprintf("/connect/v1/environments/%s/clusters/%s/connectors", environmentId, kafkaClusterId)
	err = mockProvider.AddInteraction().
		GivenWithParameter(models.ProviderState{
			Name: "connector exists",
			Parameters: map[string]interface{}{
				"account_id":       environmentId,
				"kafka_cluster_id": kafkaClusterId,
				"connector_id":     logicalClusterId,
				"connector_name":   connectorName,
			},
		}).
		UponReceiving("A request to read all connectors").
		WithRequest("GET", url, func(b *consumer.V4RequestBuilder) {
			b.
				Query("expand", m.Equality("info,status,id")).
				Header("X-Principal", m.Like("User 12345"))
		}).
		WillRespondWith(200, func(b *consumer.V4ResponseBuilder) {
			b.
				Header("Content-Type", m.Regex("application/json; charset=utf-8", "^application\\/json.*")).
				JSONBody(m.StructMatcher{
					connectorName: m.StructMatcher{
						"id": m.Map{
							// "Like" matcher checks the type of the field, but not the value
							// because our code doesn't care if the id is the same as the one in the request or not
							"id": m.Like(logicalClusterId),
						},
						"info": m.StructMatcher{
							"config": m.Map{
								// We don't care about the config values but include some just to trigger the code path;
								// the test will fail if the provider doesn't include connector.class in the response
								// TODO: how can we express "map with arbitrary string values" in http pact?
								"connector.class": m.Like("HttpSource"),
							},
						},
						"status": m.StructMatcher{
							"type": m.Like("source"),
							"connector": m.MapMatcher{
								"state": m.Like("RUNNING"),
							},
						},
					},
				})
		}).
		ExecuteTest(t, func(config consumer.MockServerConfig) error {
			connectCfg := connect.NewConfiguration()
			connectCfg.HTTPClient = &http.Client{}
			connectCfg.Servers[0].URL = fmt.Sprintf("http://%v:%v", config.Host, config.Port)
			connectCfg.AddDefaultHeader("X-Principal", "User 12345")
			client := &Client{
				connectClient: connect.NewAPIClient(connectCfg),
			}
			data := connectorResource().Data(nil)
			data.SetId("whatever") // this seems to be required
			result, err := readConnectorAndSetAttributes(context.Background(), data, client, connectorName, environmentId, kafkaClusterId)
			require.NoError(t, err)
			require.Equal(t, logicalClusterId, result[0].Id())
			return nil
		})
	require.NoError(t, err)
}
