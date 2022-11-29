package provider

import (
	kafkarestv3 "github.com/confluentinc/ccloud-sdk-go-v2/kafkarest/v3"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
)

type KafkaRestClientFactory struct {
	userAgent  string
	maxRetries *int
}

func (f KafkaRestClientFactory) CreateKafkaRestClient(restEndpoint, clusterId, clusterApiKey, clusterApiSecret string, isMetadataSetInProviderBlock bool) *KafkaRestClient {
	config := kafkarestv3.NewConfiguration()
	config.Servers[0].URL = restEndpoint
	config.UserAgent = f.userAgent
	if f.maxRetries != nil {
		config.HTTPClient = NewRetryableClientFactory(WithMaxRetries(*f.maxRetries)).CreateRetryableClient()
	} else {
		config.HTTPClient = NewRetryableClientFactory().CreateRetryableClient()
	}
	return &KafkaRestClient{
		apiClient:                    kafkarestv3.NewAPIClient(config),
		clusterId:                    clusterId,
		clusterApiKey:                clusterApiKey,
		clusterApiSecret:             clusterApiSecret,
		restEndpoint:                 restEndpoint,
		isMetadataSetInProviderBlock: isMetadataSetInProviderBlock,
	}
}

type RetryableClientFactoryOption = func(c *RetryableClientFactory)

type RetryableClientFactory struct {
	maxRetries *int
}

func WithMaxRetries(maxRetries int) RetryableClientFactoryOption {
	return func(c *RetryableClientFactory) {
		c.maxRetries = &maxRetries
	}
}

func NewRetryableClientFactory(opts ...RetryableClientFactoryOption) *RetryableClientFactory {
	c := &RetryableClientFactory{}
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

	if f.maxRetries != nil {
		retryClient.RetryMax = *f.maxRetries
	}

	return retryClient.StandardClient()
}
