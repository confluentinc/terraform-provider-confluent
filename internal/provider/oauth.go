package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	paramOAuthBlockName                = "oauth"
	paramOAuthExternalAccessToken      = "oauth_external_access_token"
	paramOAuthExternalClientId         = "oauth_external_client_id"
	paramOAuthExternalClientSecret     = "oauth_external_client_secret"
	paramOAuthExternalTokenScope       = "oauth_external_token_scope"
	paramOAuthExternalTokenURL         = "oauth_external_token_url"
	paramOAuthIdentityPoolId           = "oauth_identity_pool_id"
	paramOAuthSTSTokenExpiredInSeconds = "oauth_sts_token_expired_in_seconds"
	stsEndpoint                        = "https://api.confluent.cloud/sts/v1/oauth2/token"
)

const (
	externalTokenExpirationBuffer = 3 * time.Minute
	stsTokenExpirationBuffer      = 1 * time.Minute
)

type OAuthToken struct {
	ClientId         string       `json:"client_id"`
	ClientSecret     string       `json:"client_secret"`
	TokenUrl         string       `json:"token_url"`
	ExpiresInSeconds string       `json:"expires_in_seconds"`
	Scope            string       `json:"scope"`
	AccessToken      string       `json:"access_token"`
	TokenType        string       `json:"token_type"`
	IdentityPoolId   string       `json:"identity_pool_id"`
	ValidUntil       time.Time    `json:"valid_until"`
	HTTPClient       *http.Client `json:"http_client"`
}

type STSToken struct {
	ExpiresInSeconds string       `json:"expires_in_seconds"`
	AccessToken      string       `json:"access_token"`
	TokenType        string       `json:"token_type"`
	IssuedTokenType  string       `json:"issued_token_type"`
	IdentityPoolId   string       `json:"identity_pool_id"`
	ValidUntil       time.Time    `json:"valid_until"`
	HTTPClient       *http.Client `json:"http_client"`
}

func fetchSTSOAuthToken(ctx context.Context, subjectToken, identityPoolId, expiredInSeconds string, currToken *STSToken, retryableClient *http.Client) (*STSToken, error) {
	// Validate if the current token is still valid, if so, return it
	if valid := validateCurrentSTSOAuthToken(ctx, currToken); valid {
		return currToken, nil
	}
	return requestNewSTSOAuthToken(ctx, subjectToken, identityPoolId, expiredInSeconds, retryableClient)
}

// TODO: Try to make it as a SDK instead of raw HTTP request, tracked by: https://confluentinc.atlassian.net/browse/CLI-3511
func requestNewSTSOAuthToken(ctx context.Context, subjectToken, identityPoolId, expiredInSeconds string, retryableClient *http.Client) (*STSToken, error) {
	if retryableClient == nil {
		return nil, fmt.Errorf("retryable HTTP client is nil, cannot request new STS OAuth token")
	}

	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	data.Set("subject_token", subjectToken)
	data.Set("identity_pool_id", identityPoolId)
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt")
	data.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("expires_in", expiredInSeconds)

	req, err := http.NewRequest(http.MethodPost, stsEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	// For debug request purpose
	dumpRequest, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("Exchange STS token raw request is: %s\n", string(dumpRequest)))

	resp, err := retryableClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	// For debug response purpose
	dumpResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched STS token raw response is: %s\n", string(dumpResponse)))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("STS exchange request failed: %s", string(dumpResponse))
	}

	// Parse the response
	result := &STSToken{}
	resultMap := make(map[string]any)
	responseBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		return nil, err
	}

	for k, v := range resultMap {
		switch k {
		case "expires_in":
			result.ExpiresInSeconds = fmt.Sprintf("%v", v)
			// Be careful about the token expiry time, use half the expiry time as buffer if expiry is too short
			expiryDuration := time.Duration(v.(float64)) * time.Second
			buffer := stsTokenExpirationBuffer
			if expiryDuration <= buffer {
				buffer = expiryDuration / 2
			}
			result.ValidUntil = time.Now().Add(expiryDuration - buffer)
		case "access_token":
			result.AccessToken = v.(string)
		case "token_type":
			result.TokenType = v.(string)
		case "issued_token_type":
			result.IssuedTokenType = v.(string)
		}
	}
	result.IdentityPoolId = identityPoolId
	result.HTTPClient = retryableClient
	return result, nil
}

func fetchExternalOAuthToken(ctx context.Context, tokenUrl, clientId, clientSecret, customScope, identityPoolId string, currToken *OAuthToken, retryableClient *http.Client) (*OAuthToken, error) {
	// Validate if the current token is still valid, if so, return it
	if valid := validateCurrentExternalOAuthToken(ctx, currToken); valid {
		return currToken, nil
	}
	return requestNewExternalOAuthToken(ctx, tokenUrl, clientId, clientSecret, customScope, identityPoolId, retryableClient)
}

func requestNewExternalOAuthToken(ctx context.Context, tokenUrl, clientId, clientSecret, customScope, identityPoolId string, retryableClient *http.Client) (*OAuthToken, error) {
	if retryableClient == nil {
		return nil, fmt.Errorf("retryable HTTP client is nil, cannot request new external OAuth token")
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", clientId)
	data.Set("client_secret", clientSecret)
	if customScope != "" {
		data.Set("scope", customScope)
	}

	req, err := http.NewRequest(http.MethodPost, tokenUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")

	// For debug request purpose
	dumpRequest, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("External OAuth token raw request is: %s\n", string(dumpRequest)))

	resp, err := retryableClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	// For debug response purpose
	dumpResponse, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("External OAuth token raw response is: %s\n", string(dumpResponse)))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("exchange external token request failed: %s\n", string(dumpResponse))
	}

	// Parse the response
	result := &OAuthToken{}
	resultMap := make(map[string]any)
	responseBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(responseBody, &resultMap); err != nil {
		return nil, err
	}

	for k, v := range resultMap {
		switch k {
		case "expires_in":
			result.ExpiresInSeconds = fmt.Sprintf("%v", v)
			// Be careful about the token expiry time, use half the expiry time as buffer if expiry is too short
			expiryDuration := time.Duration(v.(float64)) * time.Second
			buffer := externalTokenExpirationBuffer
			if expiryDuration <= buffer {
				buffer = expiryDuration / 2
			}
			result.ValidUntil = time.Now().Add(expiryDuration - buffer)
		case "access_token":
			result.AccessToken = v.(string)
		case "token_type":
			result.TokenType = v.(string)
		case "scope":
			result.Scope = v.(string)
		}
	}

	result.IdentityPoolId = identityPoolId
	result.TokenUrl = tokenUrl
	result.ClientId = clientId
	result.ClientSecret = clientSecret
	result.HTTPClient = retryableClient
	return result, nil
}

func validateCurrentExternalOAuthToken(ctx context.Context, token *OAuthToken) bool {
	if token == nil || token.ValidUntil.IsZero() {
		return false
	}
	if token.ValidUntil.Before(time.Now()) {
		tflog.Info(ctx, fmt.Sprintf("Current external OAuth token expired at %s", token.ValidUntil))
		return false
	}
	return true
}

func validateCurrentSTSOAuthToken(ctx context.Context, token *STSToken) bool {
	if token == nil || token.ValidUntil.IsZero() {
		return false
	}
	if token.ValidUntil.Before(time.Now()) {
		tflog.Info(ctx, fmt.Sprintf("Current STS OAuth token expired at %s", token.ValidUntil))
		return false
	}
	return true
}

func resourceCredentialBlockValidationWithOAuth(_ context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if meta.(*Client).isOAuthEnabled && diff.HasChange(paramCredentials) {
		return fmt.Errorf("error: please remove resource credentials block when OAuth is enabled")
	}
	return nil
}

func dataSourceCredentialBlockValidationWithOAuth(d *schema.ResourceData, oauthEnabled bool) error {
	if oauthEnabled {
		if _, ok := d.GetOk(paramCredentials); ok {
			return fmt.Errorf(
				"`%s` block cannot be used when OAuth is enabled in the provider. Please remove the `credentials` block or disable OAuth",
				paramCredentials,
			)
		}
	}
	return nil
}
