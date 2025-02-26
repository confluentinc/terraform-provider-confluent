package provider

import (
	"context"
	"fmt"

	dc "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"net/http"

	fgb "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	schemaregistry "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
)

type FlinkRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f FlinkRestClientFactory) CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret string, isMetadataSetInProviderBlock bool) *FlinkRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := fgb.NewConfiguration()

	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint
	config.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()

	return &FlinkRestClient{
		apiClient:                    fgb.NewAPIClient(config),
		organizationId:               organizationId,
		environmentId:                environmentId,
		computePoolId:                computePoolId,
		principalId:                  principalId,
		flinkApiKey:                  flinkApiKey,
		flinkApiSecret:               flinkApiSecret,
		restEndpoint:                 restEndpoint,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
	}
}

type SchemaRegistryRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f SchemaRegistryRestClientFactory) CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool) *SchemaRegistryRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}

	// Setup SR API Client
	config := schemaregistry.NewConfiguration()
	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint
	config.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()

	// Setup DC API Client
	dataCatalogConfig := dc.NewConfiguration()
	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	dataCatalogConfig.UserAgent = f.userAgent
	dataCatalogConfig.Servers[0].URL = restEndpoint
	dataCatalogConfig.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()

	return &SchemaRegistryRestClient{
		apiClient:                    schemaregistry.NewAPIClient(config),
		clusterId:                    clusterId,
		clusterApiKey:                clusterApiKey,
		clusterApiSecret:             clusterApiSecret,
		restEndpoint:                 restEndpoint,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
	}
}

type CatalogRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f CatalogRestClientFactory) CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool) *CatalogRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}

	// Setup DC API Client
	dataCatalogConfig := dc.NewConfiguration()
	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	dataCatalogConfig.UserAgent = f.userAgent
	dataCatalogConfig.Servers[0].URL = restEndpoint
	dataCatalogConfig.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()

	return &CatalogRestClient{
		apiClient:                    dc.NewAPIClient(dataCatalogConfig),
		clusterId:                    clusterId,
		clusterApiKey:                clusterApiKey,
		clusterApiSecret:             clusterApiSecret,
		restEndpoint:                 restEndpoint,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
	}
}

type KafkaRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f KafkaRestClientFactory) CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock, isClusterIdSetInProviderBlock bool) *KafkaRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := kafkarestv3.NewConfiguration()

	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint
	config.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()

	return &KafkaRestClient{
		apiClient:                     kafkarestv3.NewAPIClient(config),
		clusterId:                     clusterId,
		clusterApiKey:                 clusterApiKey,
		clusterApiSecret:              clusterApiSecret,
		restEndpoint:                  restEndpoint,
		isMetadataSetInProviderBlock:  isMetadataSetInProviderBlock,
		isClusterIdSetInProviderBlock: isClusterIdSetInProviderBlock,
	}
}

type RetryableClientFactoryOption = func(c *RetryableClientFactory)

type RetryableClientFactory struct {
	ctx        context.Context
	maxRetries *int
}

func WithMaxRetries(maxRetries int) RetryableClientFactoryOption {
	return func(c *RetryableClientFactory) {
		c.maxRetries = &maxRetries
	}
}

func NewRetryableClientFactory(ctx context.Context, opts ...RetryableClientFactoryOption) *RetryableClientFactory {
	c := &RetryableClientFactory{
		ctx: ctx,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// CreateRetryableClient creates retryable HTTP client that performs automatic retries with exponential backoff for 429
// and 5** (except 501) errors. Otherwise, the response is returned and left to the caller to interpret.
func (f RetryableClientFactory) CreateRetryableClient() *http.Client {
	// Implicitly using default retry configuration
	// under the assumption is it's OK to spend retrying a single HTTP call around 15 seconds in total: 1 + 2 + 4 + 8
	// An exponential backoff equation: https://github.com/hashicorp/go-retryablehttp/blob/master/client.go#L493
	// retryWaitMax = math.Pow(2, float64(attemptNum)) * float64(retryWaitMin)
	// defaultRetryWaitMin = 1 * time.Second
	// defaultRetryWaitMax = 30 * time.Second
	// defaultRetryMax     = 4

	retryClient := retryablehttp.NewClient()
	logger := retryClientLogger{f.ctx}

	if f.maxRetries != nil {
		retryClient.RetryMax = *f.maxRetries
	}

	// Create a logger for retryablehttp
	// This logger will be used to send retryablehttp's internal logs to tflog
	retryClient.Logger = logger

	return retryClient.StandardClient()
}

// Logger is used to log messages from retryablehttp.Client to tflog.
type retryClientLogger struct {
	ctx context.Context
}

func (l retryClientLogger) Error(msg string, keysAndValues ...interface{}) {
	tflog.Error(l.ctx, msg, l.additionalFields(keysAndValues))
}

func (l retryClientLogger) Info(msg string, keysAndValues ...interface{}) {
	tflog.Info(l.ctx, msg, l.additionalFields(keysAndValues))
}

func (l retryClientLogger) Debug(msg string, keysAndValues ...interface{}) {
	tflog.Debug(l.ctx, msg, l.additionalFields(keysAndValues))
}

func (l retryClientLogger) Warn(msg string, keysAndValues ...interface{}) {
	tflog.Warn(l.ctx, msg, l.additionalFields(keysAndValues))
}

func (l retryClientLogger) additionalFields(keysAndValues []interface{}) map[string]interface{} {
	additionalFields := make(map[string]interface{}, len(keysAndValues))

	for i := 0; i+1 < len(keysAndValues); i += 2 {
		additionalFields[fmt.Sprint(keysAndValues[i])] = keysAndValues[i+1]
	}

	return additionalFields
}
