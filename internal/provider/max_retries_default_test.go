package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const maxRetriesEnvVar = "TF_PROVIDER_CONFLUENT_MAX_RETRIES"

// configureWithMaxRetriesEnv configures the provider with no max_retries in the config and the
// TF_PROVIDER_CONFLUENT_MAX_RETRIES environment variable set to env ("" means unset).
func configureWithMaxRetriesEnv(t *testing.T, env string) (*schema.ResourceData, *Client) {
	t.Helper()
	// Opt out so configuring on the default endpoint doesn't start a real telemetry transport.
	restorePublishedTelemetry(t)
	t.Setenv(disableProviderAnalyticsEnvVar, "1")
	t.Setenv(maxRetriesEnvVar, env)
	p := New("test", "")()
	d := schema.TestResourceDataRaw(t, p.Schema, map[string]interface{}{
		"cloud_api_key":    "test-key",
		"cloud_api_secret": "test-secret",
	})
	meta, diags := providerConfigure(context.Background(), d, p, "test", "")
	if diags.HasError() {
		t.Fatalf("providerConfigure failed: %v", diags)
	}
	return d, meta.(*Client)
}

func retryMaxOf(t *testing.T, name string, httpClient *http.Client) int {
	t.Helper()
	logging, ok := httpClient.Transport.(*loggingTransport)
	if !ok {
		t.Fatalf("%s client: expected a *loggingTransport, got %T", name, httpClient.Transport)
	}
	roundTripper, ok := logging.transport.(*retryablehttp.RoundTripper)
	if !ok {
		t.Fatalf("%s client: expected a *retryablehttp.RoundTripper, got %T", name, logging.transport)
	}
	return roundTripper.Client.RetryMax
}

func TestProviderMaxRetriesDefaultsToSix(t *testing.T) {
	d, c := configureWithMaxRetriesEnv(t, "")
	if got := d.Get("max_retries").(int); got != 6 {
		t.Fatalf("max_retries = %d, want 6", got)
	}
	for name, httpClient := range map[string]*http.Client{
		"Org": c.orgV2Client.GetConfig().HTTPClient,
		"CMK": c.cmkV2Client.GetConfig().HTTPClient,
	} {
		if got := retryMaxOf(t, name, httpClient); got != 6 {
			t.Errorf("%s client: RetryMax = %d, want 6", name, got)
		}
	}
	if got := *c.kafkaRestClientFactory.maxRetries; got != 6 {
		t.Errorf("Kafka REST client factory: maxRetries = %d, want 6", got)
	}
	// The rate-limited APIs keep their higher floor.
	if got := retryMaxOf(t, "IAM", c.iamV2Client.GetConfig().HTTPClient); got != iamAndRBACMaxRetriesFloor {
		t.Errorf("IAM client: RetryMax = %d, want %d", got, iamAndRBACMaxRetriesFloor)
	}
}

func TestProviderMaxRetriesFromEnvironmentVariable(t *testing.T) {
	// 4, the previous default, is still allowed.
	d, c := configureWithMaxRetriesEnv(t, "4")
	if got := d.Get("max_retries").(int); got != 4 {
		t.Fatalf("max_retries = %d, want 4 from %s", got, maxRetriesEnvVar)
	}
	if got := retryMaxOf(t, "Org", c.orgV2Client.GetConfig().HTTPClient); got != 4 {
		t.Errorf("Org client: RetryMax = %d, want 4", got)
	}

	validate := New("test", "")().Schema["max_retries"].ValidateFunc
	if _, errs := validate(4, "max_retries"); len(errs) > 0 {
		t.Errorf("max_retries = 4: unexpected validation errors %v", errs)
	}
	if _, errs := validate(3, "max_retries"); len(errs) == 0 {
		t.Error("max_retries = 3: expected a validation error")
	}
}
