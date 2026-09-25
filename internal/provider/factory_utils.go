package provider

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	datacatalogv1 "github.com/confluentinc/ccloud-sdk-go-v2/data-catalog/v1"
	flinkgatewayv1 "github.com/confluentinc/ccloud-sdk-go-v2/flink-gateway/v1"
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	schemaregistryv1 "github.com/confluentinc/ccloud-sdk-go-v2/schema-registry/v1"
	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

type FlinkRestClientFactory struct {
	ctx        context.Context
	userAgent  string
	maxRetries *int
}

func (f FlinkRestClientFactory) CreateFlinkRestClient(restEndpoint, organizationId, environmentId, computePoolId, principalId, flinkApiKey, flinkApiSecret string, isMetadataSetInProviderBlock bool, token *OAuthToken) *FlinkRestClient {
	var opts []RetryableClientFactoryOption = []RetryableClientFactoryOption{}
	config := flinkgatewayv1.NewConfiguration()

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
		apiClient:                    flinkgatewayv1.NewAPIClient(config),
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
	config := schemaregistryv1.NewConfiguration()
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
		apiClient:                    schemaregistryv1.NewAPIClient(config),
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
	dataCatalogConfig := datacatalogv1.NewConfiguration()
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
		apiClient:                    datacatalogv1.NewAPIClient(dataCatalogConfig),
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
	config := tableflowv1.NewConfiguration()

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
		apiClient:                    tableflowv1.NewAPIClient(config),
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

// CreateRetryableClient creates retryable HTTP client that performs automatic retries with
// exponential backoff for 429 and 5** (except 501) errors. Otherwise, the response is returned
// and left to the caller to interpret.
func (f RetryableClientFactory) CreateRetryableClient() *http.Client {
	// Implicitly using default retry configuration
	// under the assumption is it's OK to spend retrying a single HTTP call around 15 seconds in total: 1 + 2 + 4 + 8
	// An exponential backoff equation: https://github.com/hashicorp/go-retryablehttp/blob/master/client.go#L493
	// retryWaitMax = math.Pow(2, float64(attemptNum)) * float64(retryWaitMin)
	// defaultRetryWaitMin = 1 * time.Second
	// defaultRetryWaitMax = 30 * time.Second
	// defaultRetryMax     = 4

	retryClient := retryablehttp.NewClient()
	retryClient.Backoff = retryBackoff
	logger := retryClientLogger{f.ctx}

	if f.maxRetries != nil {
		retryClient.RetryMax = *f.maxRetries
	}

	retryClient.ErrorHandler = customErrorHandler

	// Create a logger for retryablehttp
	// This logger will be used to send retryablehttp's internal logs to tflog
	retryClient.Logger = logger

	standardClient := retryClient.StandardClient()

	standardClient.Transport = &loggingTransport{
		transport: standardClient.Transport,
		ctx:       f.ctx,
	}

	return standardClient
}

// retryBackoff scopes jitter to 429s specifically: a 429 means many callers are sharing (and just
// exceeded) a rate limit, so many confluent_connector (or other) resources can hit it at the same
// moment and, on deterministic backoff, retry in lockstep and re-trip the same limit (INC-13517).
// Every other retryable status (5xx) is left on the SDK's unmodified DefaultBackoff, so this
// change's blast radius is scoped to the failure class it's actually fixing.
func retryBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		return minFloorFullJitterBackoff(min, max, attemptNum, resp)
	}
	return retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
}

// minFloorFullJitterBackoff applies AWS's "full jitter" pattern on top of DefaultBackoff's capped
// exponential envelope, floored at min: the wait is a uniform random draw in [min, capWait]
// instead of always being exactly capWait, so concurrent callers retrying the same 429 at the same
// attempt number don't all land on the same instant. capWait's ceiling is exactly today's
// production wait (same min/max/RetryMax), so the worst case is unchanged from not jittering at
// all — only the average case improves.
//
// The floor at min (rather than 0) is deliberate: a wait close to 0s gives a rate limiter no real
// time to recover, so it's very likely to fail again immediately, wasting one of a limited number
// of retry attempts for no real chance of success rather than trading anything meaningful for the
// extra spread. The cost is that attempt 1 (attemptNum=0) is deterministic — capWait there already
// equals min, so [min, capWait] collapses to a single point and jitter only starts contributing
// from attempt 2 onward.
//
// Caveats:
//   - This assumes the API never returns a Retry-After header on a 429. If it does, DefaultBackoff
//     already returns that value verbatim as capWait, and jittering it (or, below, clamping it up
//     to min) here would not honor the server's exact instruction — this would need to check for
//     that header itself and return it as-is before falling through to this logic.
//   - Whether min=1s is actually long enough for the real rate limiter to recover is unverified;
//     see RETRY-BACKOFF-COMPARISON.md's Open Items.
func minFloorFullJitterBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	capWait := retryablehttp.DefaultBackoff(min, max, attemptNum, resp)
	if capWait <= min {
		return capWait
	}
	return min + time.Duration(rand.Int63n(int64(capWait-min)+1))
}

func customErrorHandler(resp *http.Response, err error, retries int) (*http.Response, error) {
	if resp != nil {
		body := resp.Body
		defer body.Close()
		if resp.StatusCode == 429 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 429 Too Many Requests")
		} else if resp.StatusCode == 500 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 500 Internal Server Error")
		} else if resp.StatusCode == 501 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 501 Not Implemented")
		} else if resp.StatusCode == 502 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 502 Bad Gateway")
		} else if resp.StatusCode == 503 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 503 Service Unavailable")
		} else if resp.StatusCode == 504 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 504 Gateway Timeout")
		} else if resp.StatusCode == 505 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 505 HTTP Version Not Supported")
		} else if resp.StatusCode > 505 && resp.StatusCode < 600 {
			return customErrorHandlerCode(resp, err, retries, "received HTTP 5xx error")
		}
	}
	return resp, err
}

func customErrorHandlerCode(resp *http.Response, err error, retries int, text string) (*http.Response, error) {
	if err == nil {
		return resp, fmt.Errorf(text+": (URL: %s, Method: %s, Retries: %d)", resp.Request.URL, resp.Request.Method, retries)
	} else {
		return resp, fmt.Errorf(text+": %v (URL: %s, Method: %s, Retries: %d)", err, resp.Request.URL, resp.Request.Method, retries)
	}
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

type loggingTransport struct {
	transport http.RoundTripper
	ctx       context.Context
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Call the original transport to do the actual HTTP work
	resp, err := t.transport.RoundTrip(req)

	// 2. Add our logging logic on top to output request_id for HTTP requests
	if err == nil && resp != nil && resp.Header != nil && req.URL != nil {
		if requestID := resp.Header.Get("x-request-id"); requestID != "" {
			tflog.Debug(t.ctx, "API request completed",
				map[string]interface{}{
					"request_id": requestID,
					"method":     req.Method,
					"url":        req.URL.String(),
					"status":     resp.StatusCode,
				})
		}
	}

	// 3. Return the original response
	return resp, err
}
