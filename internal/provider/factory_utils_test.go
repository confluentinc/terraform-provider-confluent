package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// newMockResponse builds an *http.Response with the given status code, a no-op
// body, and a request stub so that customErrorHandler can safely dereference
// resp.Request.URL and resp.Request.Method.
func newMockResponse(statusCode int) *http.Response {
	req, _ := http.NewRequest(http.MethodGet, "https://example.com/test", nil)
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}
}

// ---------------------------------------------------------------------------
// customErrorHandler
// ---------------------------------------------------------------------------

func TestCustomErrorHandler(t *testing.T) {
	tests := []struct {
		name        string
		resp        *http.Response
		inputErr    error
		retries     int
		wantErr     bool
		errContains string
		wantNilResp bool
		wantStatus  int // expected StatusCode on returned resp (ignored when wantNilResp)
	}{
		{
			name:        "429 Too Many Requests returns error",
			resp:        newMockResponse(429),
			inputErr:    nil,
			retries:     3,
			wantErr:     true,
			errContains: "received HTTP 429 Too Many Requests",
			wantStatus:  429,
		},
		{
			name:        "500 Internal Server Error returns error",
			resp:        newMockResponse(500),
			inputErr:    nil,
			retries:     2,
			wantErr:     true,
			errContains: "received HTTP 500 Internal Server Error",
			wantStatus:  500,
		},
		{
			name:        "501 Not Implemented returns error",
			resp:        newMockResponse(501),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "received HTTP 501 Not Implemented",
			wantStatus:  501,
		},
		{
			name:        "502 Bad Gateway returns error",
			resp:        newMockResponse(502),
			inputErr:    nil,
			retries:     0,
			wantErr:     true,
			errContains: "received HTTP 502 Bad Gateway",
			wantStatus:  502,
		},
		{
			name:        "503 Service Unavailable returns error",
			resp:        newMockResponse(503),
			inputErr:    nil,
			retries:     4,
			wantErr:     true,
			errContains: "received HTTP 503 Service Unavailable",
			wantStatus:  503,
		},
		{
			name:        "504 Gateway Timeout returns error",
			resp:        newMockResponse(504),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "received HTTP 504 Gateway Timeout",
			wantStatus:  504,
		},
		{
			name:        "505 HTTP Version Not Supported returns error",
			resp:        newMockResponse(505),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "received HTTP 505 HTTP Version Not Supported",
			wantStatus:  505,
		},
		{
			name:        "506 returns generic 5xx error",
			resp:        newMockResponse(506),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "received HTTP 5xx error",
			wantStatus:  506,
		},
		{
			name:        "599 returns generic 5xx error",
			resp:        newMockResponse(599),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "received HTTP 5xx error",
			wantStatus:  599,
		},
		{
			name:       "200 OK passes through without error",
			resp:       newMockResponse(200),
			inputErr:   nil,
			retries:    0,
			wantErr:    false,
			wantStatus: 200,
		},
		{
			name:       "201 Created passes through without error",
			resp:       newMockResponse(201),
			inputErr:   nil,
			retries:    0,
			wantErr:    false,
			wantStatus: 201,
		},
		{
			name:       "404 Not Found passes through without error",
			resp:       newMockResponse(404),
			inputErr:   nil,
			retries:    0,
			wantErr:    false,
			wantStatus: 404,
		},
		{
			name:       "400 Bad Request passes through without error",
			resp:       newMockResponse(400),
			inputErr:   nil,
			retries:    0,
			wantErr:    false,
			wantStatus: 400,
		},
		{
			name:        "nil response returns nil resp and original error",
			resp:        nil,
			inputErr:    errors.New("connection refused"),
			retries:     2,
			wantErr:     true,
			errContains: "connection refused",
			wantNilResp: true,
		},
		{
			name:        "nil response and nil error returns both nil",
			resp:        nil,
			inputErr:    nil,
			retries:     0,
			wantErr:     false,
			wantNilResp: true,
		},
		{
			name:        "429 with existing error includes original error text",
			resp:        newMockResponse(429),
			inputErr:    errors.New("upstream timeout"),
			retries:     5,
			wantErr:     true,
			errContains: "upstream timeout",
			wantStatus:  429,
		},
		{
			name:        "500 error message includes retry count",
			resp:        newMockResponse(500),
			inputErr:    nil,
			retries:     7,
			wantErr:     true,
			errContains: "Retries: 7",
			wantStatus:  500,
		},
		{
			name:        "500 error message includes URL",
			resp:        newMockResponse(500),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "https://example.com/test",
			wantStatus:  500,
		},
		{
			name:        "500 error message includes HTTP method",
			resp:        newMockResponse(500),
			inputErr:    nil,
			retries:     1,
			wantErr:     true,
			errContains: "GET",
			wantStatus:  500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, gotErr := customErrorHandler(tt.resp, tt.inputErr, tt.retries)

			if tt.wantNilResp {
				if gotResp != nil {
					t.Errorf("expected nil response, got status %d", gotResp.StatusCode)
				}
			} else {
				if gotResp == nil {
					t.Fatal("expected non-nil response, got nil")
				}
				if gotResp.StatusCode != tt.wantStatus {
					t.Errorf("expected status %d, got %d", tt.wantStatus, gotResp.StatusCode)
				}
			}

			if tt.wantErr {
				if gotErr == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(gotErr.Error(), tt.errContains) {
					t.Errorf("expected error to contain %q, got %q", tt.errContains, gotErr.Error())
				}
			} else {
				if gotErr != nil {
					t.Errorf("expected no error, got %v", gotErr)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// customErrorHandlerCode
// ---------------------------------------------------------------------------

func TestCustomErrorHandlerCode(t *testing.T) {
	tests := []struct {
		name     string
		inputErr error
		retries  int
		text     string
		wantErr  string
	}{
		{
			name:     "nil err produces message without wrapped error",
			inputErr: nil,
			retries:  3,
			text:     "received HTTP 500 Internal Server Error",
			wantErr:  "received HTTP 500 Internal Server Error: (URL: https://example.com/test, Method: GET, Retries: 3)",
		},
		{
			name:     "non-nil err is included in message",
			inputErr: errors.New("something broke"),
			retries:  2,
			text:     "received HTTP 429 Too Many Requests",
			wantErr:  "received HTTP 429 Too Many Requests: something broke (URL: https://example.com/test, Method: GET, Retries: 2)",
		},
		{
			name:     "zero retries",
			inputErr: nil,
			retries:  0,
			text:     "received HTTP 502 Bad Gateway",
			wantErr:  "received HTTP 502 Bad Gateway: (URL: https://example.com/test, Method: GET, Retries: 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := newMockResponse(500)
			gotResp, gotErr := customErrorHandlerCode(resp, tt.inputErr, tt.retries, tt.text)

			if gotResp != resp {
				t.Error("expected the same response object to be returned")
			}
			if gotErr == nil {
				t.Fatal("expected an error, got nil")
			}
			if gotErr.Error() != tt.wantErr {
				t.Errorf("error mismatch:\n  want: %s\n  got:  %s", tt.wantErr, gotErr.Error())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// additionalFields
// ---------------------------------------------------------------------------

func TestAdditionalFields(t *testing.T) {
	logger := retryClientLogger{ctx: context.Background()}

	tests := []struct {
		name          string
		keysAndValues []interface{}
		wantLen       int
		wantPairs     map[string]interface{}
	}{
		{
			name:          "empty list",
			keysAndValues: []interface{}{},
			wantLen:       0,
			wantPairs:     map[string]interface{}{},
		},
		{
			name:          "single pair",
			keysAndValues: []interface{}{"key1", "value1"},
			wantLen:       1,
			wantPairs:     map[string]interface{}{"key1": "value1"},
		},
		{
			name:          "multiple pairs",
			keysAndValues: []interface{}{"a", 1, "b", 2, "c", 3},
			wantLen:       3,
			wantPairs:     map[string]interface{}{"a": 1, "b": 2, "c": 3},
		},
		{
			name:          "odd number of elements drops trailing key",
			keysAndValues: []interface{}{"key1", "value1", "orphan"},
			wantLen:       1,
			wantPairs:     map[string]interface{}{"key1": "value1"},
		},
		{
			name:          "non-string key is converted via Sprint",
			keysAndValues: []interface{}{42, "numeric-key", true, "bool-key"},
			wantLen:       2,
			wantPairs:     map[string]interface{}{"42": "numeric-key", "true": "bool-key"},
		},
		{
			name:          "nil value is preserved",
			keysAndValues: []interface{}{"key", nil},
			wantLen:       1,
			wantPairs:     map[string]interface{}{"key": nil},
		},
		{
			name:          "single element (odd) produces empty map",
			keysAndValues: []interface{}{"lonely"},
			wantLen:       0,
			wantPairs:     map[string]interface{}{},
		},
		{
			name:          "duplicate keys - later value wins",
			keysAndValues: []interface{}{"key", "first", "key", "second"},
			wantLen:       1,
			wantPairs:     map[string]interface{}{"key": "second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logger.additionalFields(tt.keysAndValues)

			if len(got) != tt.wantLen {
				t.Errorf("expected map length %d, got %d", tt.wantLen, len(got))
			}
			for k, v := range tt.wantPairs {
				gotVal, ok := got[k]
				if !ok {
					t.Errorf("expected key %q to be present", k)
					continue
				}
				if fmt.Sprint(gotVal) != fmt.Sprint(v) {
					t.Errorf("key %q: expected %v, got %v", k, v, gotVal)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// WithMaxRetries
// ---------------------------------------------------------------------------

func TestWithMaxRetries(t *testing.T) {
	tests := []struct {
		name       string
		maxRetries int
	}{
		{name: "zero retries", maxRetries: 0},
		{name: "positive retries", maxRetries: 5},
		{name: "large retries", maxRetries: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := WithMaxRetries(tt.maxRetries)
			factory := &RetryableClientFactory{}
			opt(factory)

			if factory.maxRetries == nil {
				t.Fatal("expected maxRetries to be set, got nil")
			}
			if *factory.maxRetries != tt.maxRetries {
				t.Errorf("expected maxRetries %d, got %d", tt.maxRetries, *factory.maxRetries)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NewRetryableClientFactory
// ---------------------------------------------------------------------------

func TestNewRetryableClientFactory(t *testing.T) {
	t.Run("without options", func(t *testing.T) {
		ctx := context.Background()
		factory := NewRetryableClientFactory(ctx)

		if factory == nil {
			t.Fatal("expected non-nil factory")
		}
		if factory.ctx != ctx {
			t.Error("expected context to be set")
		}
		if factory.maxRetries != nil {
			t.Errorf("expected maxRetries to be nil, got %d", *factory.maxRetries)
		}
	})

	t.Run("with WithMaxRetries option", func(t *testing.T) {
		ctx := context.Background()
		factory := NewRetryableClientFactory(ctx, WithMaxRetries(10))

		if factory == nil {
			t.Fatal("expected non-nil factory")
		}
		if factory.ctx != ctx {
			t.Error("expected context to be set")
		}
		if factory.maxRetries == nil {
			t.Fatal("expected maxRetries to be set")
		}
		if *factory.maxRetries != 10 {
			t.Errorf("expected maxRetries 10, got %d", *factory.maxRetries)
		}
	})

	t.Run("last option wins for maxRetries", func(t *testing.T) {
		ctx := context.Background()
		factory := NewRetryableClientFactory(ctx, WithMaxRetries(3), WithMaxRetries(7))

		if factory.maxRetries == nil {
			t.Fatal("expected maxRetries to be set")
		}
		if *factory.maxRetries != 7 {
			t.Errorf("expected maxRetries 7 (last option wins), got %d", *factory.maxRetries)
		}
	})
}

// ---------------------------------------------------------------------------
// customErrorHandler with POST method
// ---------------------------------------------------------------------------

func TestCustomErrorHandlerWithPostMethod(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "https://api.confluent.cloud/resource", nil)
	resp := &http.Response{
		StatusCode: 503,
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}

	_, gotErr := customErrorHandler(resp, nil, 4)
	if gotErr == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := gotErr.Error()
	if !strings.Contains(errMsg, "POST") {
		t.Errorf("expected error to contain method POST, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "https://api.confluent.cloud/resource") {
		t.Errorf("expected error to contain URL, got %q", errMsg)
	}
	if !strings.Contains(errMsg, "Retries: 4") {
		t.Errorf("expected error to contain retry count, got %q", errMsg)
	}
}

// ---------------------------------------------------------------------------
// Edge case: status code 600 should not trigger 5xx handler
// ---------------------------------------------------------------------------

func TestCustomErrorHandlerStatusCode600(t *testing.T) {
	resp := newMockResponse(600)
	gotResp, gotErr := customErrorHandler(resp, nil, 0)

	if gotErr != nil {
		t.Errorf("expected no error for status 600, got %v", gotErr)
	}
	if gotResp == nil {
		t.Fatal("expected non-nil response")
	}
	if gotResp.StatusCode != 600 {
		t.Errorf("expected status 600, got %d", gotResp.StatusCode)
	}
}
