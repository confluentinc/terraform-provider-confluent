package provider

import (
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	paramOAuthBlockName      = "oauth"
	paramOAuthAccessToken    = "oauth_access_token"
	paramOAuthClientId       = "oauth_client_id"
	paramOAuthClientSecret   = "oauth_client_secret"
	paramOAuthRefreshToken   = "oauth_refresh_token"
	paramOAuthScope          = "oauth_scope"
	paramOAuthSTSToken       = "oauth_sts_token"
	paramOAuthTokenURL       = "oauth_token_url"
	paramOAuthIdentityPoolId = "oauth_identity_pool_id"
)

type OAuthClientConfig struct {
	ExternalTokenSource oauth2.TokenSource
	STSTokenSource      oauth2.TokenSource
	ExternalHTTPClient  *http.Client
	STSHTTPClient       *http.Client
}

type STSExchange struct {
	OAuthTokenSource oauth2.TokenSource // Source for the original OAuth token
	IdentityPoolID   string
	STSURL           string        // 'STSURL' adheres to Go’s acronym conventions
	CurrentToken     *oauth2.Token // Cached STS token
}

// Token fetches a new STS token if expired
func (s *STSExchange) Token() (*oauth2.Token, error) {
	// If the token is still valid, return it
	if s.CurrentToken != nil && time.Now().Before(s.CurrentToken.Expiry) {
		return s.CurrentToken, nil
	}

	// Get a fresh OAuth token
	oauthToken, err := s.OAuthTokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth token: %v", err)
	}

	// Exchange OAuth token for STS token
	newToken, err := exchangeSTS(oauthToken.AccessToken, s.IdentityPoolID, s.STSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange STS token: %v", err)
	}

	// Cache the new STS token
	s.CurrentToken = newToken
	return newToken, nil
}

func exchangeSTS(oauthToken, identityPoolID, stsURL string) (*oauth2.Token, error) {
	data := url.Values{}
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
	data.Set("subject_token", oauthToken)
	data.Set("identity_pool_id", identityPoolID)
	data.Set("subject_token_type", "urn:ietf:params:oauth:token-type:jwt")
	data.Set("requested_token_type", "urn:ietf:params:oauth:token-type:access_token")
	data.Set("expires_in", "900")

	req, err := http.NewRequest("POST", stsURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("STS request failed: %s", string(body))
	}

	// Parse the response
	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Return STS token as an `oauth2.Token`
	return &oauth2.Token{
		AccessToken: result.AccessToken,
		Expiry:      time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}
