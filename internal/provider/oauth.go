package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	sts "github.com/confluentinc/ccloud-sdk-go-v2/sts/v1"
)

const (
	paramOAuthBlockName            = "oauth"
	paramOAuthExternalAccessToken  = "oauth_external_access_token"
	paramOAuthExternalClientId     = "oauth_external_client_id"
	paramOAuthExternalClientSecret = "oauth_external_client_secret"
	paramOAuthExternalTokenScope   = "oauth_external_token_scope"
	paramOAuthExternalTokenURL     = "oauth_external_token_url"
	paramOAuthIdentityPoolId       = "oauth_identity_pool_id"
)

const (
	paramOAuthSTSTokenExpiredInSeconds        = "oauth_sts_token_expired_in_seconds"
	paramOAuthSTSTokenGrantTypeValue          = "urn:ietf:params:oauth:grant-type:token-exchange"
	paramOAuthSTSTokenSubjectTokenTypeValue   = "urn:ietf:params:oauth:token-type:jwt"
	paramOAuthSTSTokenRequestedTokenTypeValue = "urn:ietf:params:oauth:token-type:access_token"
)

const (
	externalTokenExpirationBuffer = 3 * time.Minute
	stsTokenExpirationBuffer      = 1 * time.Minute
)

const (
	configOAuthBearer = "OAUTHBEARER"
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
	ExpiresInSeconds string         `json:"expires_in_seconds"`
	AccessToken      string         `json:"access_token"`
	TokenType        string         `json:"token_type"`
	IssuedTokenType  string         `json:"issued_token_type"`
	IdentityPoolId   string         `json:"identity_pool_id"`
	ValidUntil       time.Time      `json:"valid_until"`
	STSClient        *sts.APIClient `json:"sts_client"`
}

func fetchSTSOAuthToken(ctx context.Context, subjectToken, identityPoolId, expiredInSeconds string, currToken *STSToken, stsClient *sts.APIClient) (*STSToken, error) {
	// Validate if the current token is still valid, if so, return it
	if valid := validateCurrentSTSOAuthToken(ctx, currToken); valid {
		return currToken, nil
	}
	return requestNewSTSOAuthToken(ctx, subjectToken, identityPoolId, expiredInSeconds, stsClient)
}

func requestNewSTSOAuthToken(ctx context.Context, subjectToken, identityPoolId, expiredInSeconds string, stsClient *sts.APIClient) (*STSToken, error) {
	if stsClient == nil {
		return nil, fmt.Errorf("STS HTTP client is nil, cannot request new STS OAuth token")
	}

	req := stsClient.OAuthTokensStsV1Api.ExchangeStsV1OauthToken(ctx).
		GrantType(paramOAuthSTSTokenGrantTypeValue).
		SubjectToken(subjectToken).
		IdentityPoolId(identityPoolId).
		SubjectTokenType(paramOAuthSTSTokenSubjectTokenTypeValue).
		RequestedTokenType(paramOAuthSTSTokenRequestedTokenTypeValue)

	// Handle the optional "expires_in" string parameter
	if expiredInSeconds != "" {
		expiredInSecondsInt, err := strconv.ParseInt(expiredInSeconds, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("error casting `oauth_sts_token_expired_in_seconds` value: %s, must be a valid integer", expiredInSeconds)
		}
		req = req.ExpiresIn(int32(expiredInSecondsInt))
	}

	tflog.Debug(ctx, "requesting new STS OAuth token")

	resp, status, err := req.Execute()
	// Both Transport-level error (DNS, timeout, TLS, etc.)
	// and Application-level error (HTTP 4xx/5xx) are handled here
	if err != nil {
		if status != nil && status.Body != nil {
			body, _ := io.ReadAll(status.Body)
			return nil, fmt.Errorf("STS token exchange request failed: status=%s, description=%s", status.Status, string(body))
		}
		return nil, fmt.Errorf("STS token exchange request failed: %w", err)
	}

	// Parse the response
	result := &STSToken{}
	result.ExpiresInSeconds = strconv.Itoa(int(resp.ExpiresIn))
	result.AccessToken = resp.AccessToken
	result.TokenType = resp.TokenType
	result.IssuedTokenType = resp.IssuedTokenType

	// Be careful about the token expiry time, use half the expiry time as buffer if expiry is too short
	expiryDuration := time.Duration(resp.ExpiresIn) * time.Second
	buffer := stsTokenExpirationBuffer
	if expiryDuration <= buffer {
		buffer = expiryDuration / 2
	}
	result.ValidUntil = time.Now().Add(expiryDuration - buffer)
	result.IdentityPoolId = identityPoolId
	result.STSClient = stsClient
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

	req, err := buildExternalOAuthRequest(tokenUrl, clientId, clientSecret, customScope)
	if err != nil {
		return nil, fmt.Errorf("failed to build external OAuth request: %w", err)
	}

	tflog.Debug(ctx, "requesting new external OAuth token")

	resp, err := retryableClient.Do(req)
	// Transport-level error (DNS, timeout, TLS, etc.)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			tflog.Warn(ctx, "failed to close external token exchange response body", map[string]any{"error": closeErr.Error()})
		}
	}()

	// Application-level error (HTTP 4xx/5xx)
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read external token exchange response body: %w", err)
		}
		return nil, fmt.Errorf("external token exchange request failed: status=%s, description=%s", resp.Status, string(body))
	}

	// Parse the response
	result, err := parseExternalOAuthResponse(resp.Body, customScope, identityPoolId, tokenUrl, clientId, clientSecret, retryableClient)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OAuth response: %w", err)
	}
	tflog.Debug(ctx, "new external OAuth token acquired", map[string]any{
		"expires_in":  result.ExpiresInSeconds + "s",
		"valid_until": result.ValidUntil.Format(time.RFC3339),
	})
	return result, nil
}

func buildExternalOAuthRequest(tokenURL, clientID, clientSecret, customScope string) (*http.Request, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}
	if customScope != "" {
		data.Set("scope", customScope)
	}

	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json")
	return req, nil
}

func parseExternalOAuthResponse(body io.Reader, customScope, identityPoolId, tokenUrl, clientId, clientSecret string, client *http.Client) (*OAuthToken, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}

	var respMap map[string]any
	if err := json.Unmarshal(raw, &respMap); err != nil {
		return nil, err
	}

	result := &OAuthToken{
		Scope:          customScope,
		IdentityPoolId: identityPoolId,
		TokenUrl:       tokenUrl,
		ClientId:       clientId,
		ClientSecret:   clientSecret,
		HTTPClient:     client,
	}

	if v, ok := respMap["access_token"].(string); ok {
		result.AccessToken = v
	}
	if v, ok := respMap["token_type"].(string); ok {
		result.TokenType = v
	}
	if v, ok := respMap["expires_in"]; ok {
		result.ExpiresInSeconds = fmt.Sprintf("%v", v)
		// Be careful about the token expiry time, use half the expiry time as buffer if expiry is too short
		expiryDuration := time.Duration(v.(float64)) * time.Second
		buffer := externalTokenExpirationBuffer
		if expiryDuration <= buffer {
			buffer = expiryDuration / 2
		}
		result.ValidUntil = time.Now().Add(expiryDuration - buffer)
	}

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

func resourceCredentialBlockValidationWithOAuth(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	if meta.(*Client).isOAuthEnabled && diff.HasChange(paramCredentials) {
		// When migrating from API key/secret to OAuth authentication
		// `oldValue` in the TF state is non-empty slice, `newValue` in the TF config file is empty slice (not `nil`)
		oldValue, newValue := diff.GetChange(paramCredentials)
		if interfaceToSliceLen(oldValue) > 0 && interfaceToSliceLen(newValue) == 0 {
			tflog.Debug(ctx, fmt.Sprintf("resource credentials block removed, OAuth is enabled in the provider"))
			return nil
		}
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

func interfaceToSliceLen(v interface{}) int {
	if list, ok := v.([]interface{}); ok {
		return len(list)
	}
	return 0
}
