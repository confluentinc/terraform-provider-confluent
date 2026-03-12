package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestValidateCurrentExternalOAuthToken(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		token    *OAuthToken
		expected bool
	}{
		{
			name:     "nil token returns false",
			token:    nil,
			expected: false,
		},
		{
			name:     "zero ValidUntil returns false",
			token:    &OAuthToken{ValidUntil: time.Time{}},
			expected: false,
		},
		{
			name:     "expired token returns false",
			token:    &OAuthToken{ValidUntil: time.Now().Add(-1 * time.Hour)},
			expected: false,
		},
		{
			name:     "valid token returns true",
			token:    &OAuthToken{ValidUntil: time.Now().Add(1 * time.Hour)},
			expected: true,
		},
		{
			name:     "token expiring in 1 second still valid",
			token:    &OAuthToken{ValidUntil: time.Now().Add(1 * time.Second)},
			expected: true,
		},
		{
			name:     "token that just expired returns false",
			token:    &OAuthToken{ValidUntil: time.Now().Add(-1 * time.Millisecond)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCurrentExternalOAuthToken(ctx, tt.token)
			if result != tt.expected {
				t.Errorf("validateCurrentExternalOAuthToken() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateCurrentSTSOAuthToken(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		token    *STSToken
		expected bool
	}{
		{
			name:     "nil token returns false",
			token:    nil,
			expected: false,
		},
		{
			name:     "zero ValidUntil returns false",
			token:    &STSToken{ValidUntil: time.Time{}},
			expected: false,
		},
		{
			name:     "expired token returns false",
			token:    &STSToken{ValidUntil: time.Now().Add(-1 * time.Hour)},
			expected: false,
		},
		{
			name:     "valid token returns true",
			token:    &STSToken{ValidUntil: time.Now().Add(1 * time.Hour)},
			expected: true,
		},
		{
			name:     "token expiring in 1 second still valid",
			token:    &STSToken{ValidUntil: time.Now().Add(1 * time.Second)},
			expected: true,
		},
		{
			name:     "token that just expired returns false",
			token:    &STSToken{ValidUntil: time.Now().Add(-1 * time.Millisecond)},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCurrentSTSOAuthToken(ctx, tt.token)
			if result != tt.expected {
				t.Errorf("validateCurrentSTSOAuthToken() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildExternalOAuthRequest(t *testing.T) {
	tests := []struct {
		name         string
		tokenURL     string
		clientID     string
		clientSecret string
		customScope  string
		wantErr      bool
		checkFunc    func(t *testing.T, req *http.Request)
	}{
		{
			name:         "basic request without scope",
			tokenURL:     "https://example.com/oauth/token",
			clientID:     "my-client-id",
			clientSecret: "my-client-secret",
			customScope:  "",
			wantErr:      false,
			checkFunc: func(t *testing.T, req *http.Request) {
				if req.Method != http.MethodPost {
					t.Errorf("expected method POST, got %s", req.Method)
				}
				if req.URL.String() != "https://example.com/oauth/token" {
					t.Errorf("expected URL https://example.com/oauth/token, got %s", req.URL.String())
				}
				if ct := req.Header.Get("content-type"); ct != "application/x-www-form-urlencoded" {
					t.Errorf("expected content-type application/x-www-form-urlencoded, got %s", ct)
				}
				if accept := req.Header.Get("accept"); accept != "application/json" {
					t.Errorf("expected accept application/json, got %s", accept)
				}

				bodyBytes, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				body := string(bodyBytes)
				if !strings.Contains(body, "grant_type=client_credentials") {
					t.Error("body missing grant_type=client_credentials")
				}
				if !strings.Contains(body, "client_id=my-client-id") {
					t.Error("body missing client_id")
				}
				if !strings.Contains(body, "client_secret=my-client-secret") {
					t.Error("body missing client_secret")
				}
				if strings.Contains(body, "scope=") {
					t.Error("body should not contain scope when customScope is empty")
				}
			},
		},
		{
			name:         "request with custom scope",
			tokenURL:     "https://example.com/oauth/token",
			clientID:     "my-client-id",
			clientSecret: "my-client-secret",
			customScope:  "read write",
			wantErr:      false,
			checkFunc: func(t *testing.T, req *http.Request) {
				bodyBytes, err := io.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("failed to read request body: %v", err)
				}
				body := string(bodyBytes)
				if !strings.Contains(body, "scope=read+write") {
					t.Errorf("body missing or incorrect scope, got: %s", body)
				}
			},
		},
		{
			name:         "invalid URL returns error",
			tokenURL:     "://invalid-url",
			clientID:     "id",
			clientSecret: "secret",
			customScope:  "",
			wantErr:      true,
			checkFunc:    nil,
		},
		{
			name:         "empty client credentials still builds request",
			tokenURL:     "https://example.com/token",
			clientID:     "",
			clientSecret: "",
			customScope:  "",
			wantErr:      false,
			checkFunc: func(t *testing.T, req *http.Request) {
				bodyBytes, _ := io.ReadAll(req.Body)
				body := string(bodyBytes)
				if !strings.Contains(body, "client_id=") {
					t.Error("body missing client_id parameter")
				}
				if !strings.Contains(body, "client_secret=") {
					t.Error("body missing client_secret parameter")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := buildExternalOAuthRequest(tt.tokenURL, tt.clientID, tt.clientSecret, tt.customScope)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildExternalOAuthRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFunc != nil && req != nil {
				tt.checkFunc(t, req)
			}
		})
	}
}

func TestParseExternalOAuthResponse(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		customScope    string
		identityPoolId string
		tokenUrl       string
		clientId       string
		clientSecret   string
		wantErr        bool
		checkFunc      func(t *testing.T, token *OAuthToken)
	}{
		{
			name:           "valid response with all fields",
			body:           `{"access_token":"test-token","token_type":"Bearer","expires_in":3600}`,
			customScope:    "read",
			identityPoolId: "pool-123",
			tokenUrl:       "https://example.com/token",
			clientId:       "client-id",
			clientSecret:   "client-secret",
			wantErr:        false,
			checkFunc: func(t *testing.T, token *OAuthToken) {
				if token.AccessToken != "test-token" {
					t.Errorf("expected access_token 'test-token', got '%s'", token.AccessToken)
				}
				if token.TokenType != "Bearer" {
					t.Errorf("expected token_type 'Bearer', got '%s'", token.TokenType)
				}
				if token.ExpiresInSeconds != "3600" {
					t.Errorf("expected expires_in_seconds '3600', got '%s'", token.ExpiresInSeconds)
				}
				if token.Scope != "read" {
					t.Errorf("expected scope 'read', got '%s'", token.Scope)
				}
				if token.IdentityPoolId != "pool-123" {
					t.Errorf("expected identity_pool_id 'pool-123', got '%s'", token.IdentityPoolId)
				}
				if token.TokenUrl != "https://example.com/token" {
					t.Errorf("expected token_url 'https://example.com/token', got '%s'", token.TokenUrl)
				}
				if token.ClientId != "client-id" {
					t.Errorf("expected client_id 'client-id', got '%s'", token.ClientId)
				}
				if token.ClientSecret != "client-secret" {
					t.Errorf("expected client_secret 'client-secret', got '%s'", token.ClientSecret)
				}
				if token.ValidUntil.IsZero() {
					t.Error("expected ValidUntil to be set")
				}
				// 3600s - 3min buffer = ~3420s from now
				expectedMin := time.Now().Add(3400 * time.Second)
				expectedMax := time.Now().Add(3600 * time.Second)
				if token.ValidUntil.Before(expectedMin) || token.ValidUntil.After(expectedMax) {
					t.Errorf("ValidUntil %v not within expected range [%v, %v]", token.ValidUntil, expectedMin, expectedMax)
				}
			},
		},
		{
			name:           "valid response with short expiry uses half buffer",
			body:           `{"access_token":"short-token","token_type":"Bearer","expires_in":60}`,
			customScope:    "",
			identityPoolId: "pool-456",
			tokenUrl:       "https://example.com/token",
			clientId:       "cid",
			clientSecret:   "csec",
			wantErr:        false,
			checkFunc: func(t *testing.T, token *OAuthToken) {
				// 60s expiry is less than 3min buffer, so buffer = 60/2 = 30s
				// ValidUntil should be ~30s from now
				expectedMin := time.Now().Add(20 * time.Second)
				expectedMax := time.Now().Add(40 * time.Second)
				if token.ValidUntil.Before(expectedMin) || token.ValidUntil.After(expectedMax) {
					t.Errorf("ValidUntil %v not within expected range [%v, %v] for short expiry", token.ValidUntil, expectedMin, expectedMax)
				}
			},
		},
		{
			name:           "response without expires_in leaves ValidUntil zero",
			body:           `{"access_token":"no-expiry","token_type":"Bearer"}`,
			customScope:    "",
			identityPoolId: "",
			tokenUrl:       "https://example.com/token",
			clientId:       "cid",
			clientSecret:   "csec",
			wantErr:        false,
			checkFunc: func(t *testing.T, token *OAuthToken) {
				if token.AccessToken != "no-expiry" {
					t.Errorf("expected access_token 'no-expiry', got '%s'", token.AccessToken)
				}
				if !token.ValidUntil.IsZero() {
					t.Errorf("expected ValidUntil to be zero when expires_in is missing, got %v", token.ValidUntil)
				}
				if token.ExpiresInSeconds != "" {
					t.Errorf("expected ExpiresInSeconds to be empty, got '%s'", token.ExpiresInSeconds)
				}
			},
		},
		{
			name:           "response with missing access_token",
			body:           `{"token_type":"Bearer","expires_in":3600}`,
			customScope:    "",
			identityPoolId: "",
			tokenUrl:       "https://example.com/token",
			clientId:       "cid",
			clientSecret:   "csec",
			wantErr:        false,
			checkFunc: func(t *testing.T, token *OAuthToken) {
				if token.AccessToken != "" {
					t.Errorf("expected empty access_token, got '%s'", token.AccessToken)
				}
			},
		},
		{
			name:           "invalid JSON returns error",
			body:           `{not valid json}`,
			customScope:    "",
			identityPoolId: "",
			tokenUrl:       "https://example.com/token",
			clientId:       "cid",
			clientSecret:   "csec",
			wantErr:        true,
			checkFunc:      nil,
		},
		{
			name:           "empty body returns error",
			body:           "",
			customScope:    "",
			identityPoolId: "",
			tokenUrl:       "https://example.com/token",
			clientId:       "cid",
			clientSecret:   "csec",
			wantErr:        true,
			checkFunc:      nil,
		},
		{
			name:           "passthrough fields are set correctly",
			body:           `{"access_token":"tok"}`,
			customScope:    "my-scope",
			identityPoolId: "my-pool",
			tokenUrl:       "https://idp.example.com/token",
			clientId:       "the-client",
			clientSecret:   "the-secret",
			wantErr:        false,
			checkFunc: func(t *testing.T, token *OAuthToken) {
				if token.Scope != "my-scope" {
					t.Errorf("expected scope 'my-scope', got '%s'", token.Scope)
				}
				if token.IdentityPoolId != "my-pool" {
					t.Errorf("expected identity_pool_id 'my-pool', got '%s'", token.IdentityPoolId)
				}
				if token.TokenUrl != "https://idp.example.com/token" {
					t.Errorf("expected token_url, got '%s'", token.TokenUrl)
				}
				if token.ClientId != "the-client" {
					t.Errorf("expected client_id 'the-client', got '%s'", token.ClientId)
				}
				if token.ClientSecret != "the-secret" {
					t.Errorf("expected client_secret 'the-secret', got '%s'", token.ClientSecret)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.body)
			token, err := parseExternalOAuthResponse(reader, tt.customScope, tt.identityPoolId, tt.tokenUrl, tt.clientId, tt.clientSecret, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseExternalOAuthResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkFunc != nil && token != nil {
				tt.checkFunc(t, token)
			}
		})
	}
}

func TestInterfaceToSliceLen(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "nil returns 0",
			input:    nil,
			expected: 0,
		},
		{
			name:     "empty slice returns 0",
			input:    []interface{}{},
			expected: 0,
		},
		{
			name:     "slice with one element returns 1",
			input:    []interface{}{"hello"},
			expected: 1,
		},
		{
			name:     "slice with multiple elements",
			input:    []interface{}{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "string value returns 0",
			input:    "not a slice",
			expected: 0,
		},
		{
			name:     "integer value returns 0",
			input:    42,
			expected: 0,
		},
		{
			name:     "typed slice returns 0 (not []interface{})",
			input:    []string{"a", "b"},
			expected: 0,
		},
		{
			name:     "map returns 0",
			input:    map[string]interface{}{"key": "value"},
			expected: 0,
		},
		{
			name:     "slice with nil elements",
			input:    []interface{}{nil, nil},
			expected: 2,
		},
		{
			name:     "slice with mixed types",
			input:    []interface{}{1, "two", 3.0, true, nil},
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interfaceToSliceLen(tt.input)
			if result != tt.expected {
				t.Errorf("interfaceToSliceLen() = %d, want %d", result, tt.expected)
			}
		})
	}
}
