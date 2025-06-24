package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	dc "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	fgb "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	schemaregistry "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	tableflow "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

type FlinkRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f FlinkRestClientFactory) CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret string, isMetadataSetInProviderBlock bool, token *OAuthToken) *FlinkRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := fgb.NewConfiguration()

	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint

	baseFactory := NewRetryableClientFactory(f.ctx, opts...)

	config.HTTPClient = baseFactory.CreateRetryableClient()
	if token != nil {
		config.DefaultHeader = map[string]string{"confluent-identity-pool-id": token.IdentityPoolId}
	}

	return &FlinkRestClient{
		apiClient:                    fgb.NewAPIClient(config),
		externalAccessToken:          token,
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

func (f SchemaRegistryRestClientFactory) CreateSchemaRegistryRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool, token *OAuthToken) *SchemaRegistryRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}

	// Setup SR API Client
	config := schemaregistry.NewConfiguration()
	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint
	config.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()
	if token != nil {
		config.DefaultHeader = map[string]string{"confluent-identity-pool-id": token.IdentityPoolId, "target-sr-cluster": clusterId}
	}

	return &SchemaRegistryRestClient{
		apiClient:                    schemaregistry.NewAPIClient(config),
		externalAccessToken:          token,
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

func (f CatalogRestClientFactory) CreateCatalogRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool, token *OAuthToken) *CatalogRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}

	// Setup DC API Client
	dataCatalogConfig := dc.NewConfiguration()
	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	dataCatalogConfig.UserAgent = f.userAgent
	dataCatalogConfig.Servers[0].URL = restEndpoint
	dataCatalogConfig.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()
	if token != nil {
		dataCatalogConfig.DefaultHeader = map[string]string{"confluent-identity-pool-id": token.IdentityPoolId, "target-sr-cluster": clusterId}
	}

	return &CatalogRestClient{
		apiClient:                    dc.NewAPIClient(dataCatalogConfig),
		externalAccessToken:          token,
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

func (f KafkaRestClientFactory) CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isClusterIdSetInProviderBlock, isMetadataSetInProviderBlock bool, token *OAuthToken) *KafkaRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := kafkarestv3.NewConfiguration()

	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = restEndpoint

	baseFactory := NewRetryableClientFactory(f.ctx, opts...)

	config.HTTPClient = baseFactory.CreateRetryableClient()
	if token != nil {
		config.DefaultHeader = map[string]string{"confluent-identity-pool-id": token.IdentityPoolId}
	}

	return &KafkaRestClient{
		apiClient:                     kafkarestv3.NewAPIClient(config),
		externalAccessToken:           token,
		clusterId:                     clusterId,
		clusterApiKey:                 clusterApiKey,
		clusterApiSecret:              clusterApiSecret,
		restEndpoint:                  restEndpoint,
		isMetadataSetInProviderBlock:  isMetadataSetInProviderBlock,
		isClusterIdSetInProviderBlock: isClusterIdSetInProviderBlock,
	}
}

type TableflowRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
	endpoint   string
}

func (f TableflowRestClientFactory) CreateTableflowRestClient(tableflowApiKey, tableflowApiSecret string, isMetadataSetInProviderBlock bool, externalToken *OAuthToken, stsToken *STSToken) *TableflowRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := tableflow.NewConfiguration()

	if f.maxRetries != nil {
		opts = append(opts, WithMaxRetries(*f.maxRetries))
	}

	config.UserAgent = f.userAgent
	config.Servers[0].URL = f.endpoint
	config.HTTPClient = NewRetryableClientFactory(f.ctx, opts...).CreateRetryableClient()
	if externalToken != nil {
		config.DefaultHeader = map[string]string{"confluent-identity-pool-id": externalToken.IdentityPoolId}
	}

	return &TableflowRestClient{
		apiClient:                    tableflow.NewAPIClient(config),
		oauthToken:                   externalToken,
		stsToken:                     stsToken,
		tableflowApiKey:              tableflowApiKey,
		tableflowApiSecret:           tableflowApiSecret,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
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

	retryClient.ErrorHandler = customErrorHandler

	// Create a logger for retryablehttp
	// This logger will be used to send retryablehttp's internal logs to tflog
	retryClient.Logger = logger

	return retryClient.StandardClient()
}

func customErrorHandler(resp *http.Response, err error, _ int) (*http.Response, error) {
	if resp != nil {
		if resp.StatusCode == 429 {
			return resp, fmt.Errorf("received HTTP 429 Too Many Requests: %v (URL: %s, Method: %s)", err, resp.Request.URL, resp.Request.Method)
		}
	}
	return resp, err
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
