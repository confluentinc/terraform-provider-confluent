// Copyright 2021 Confluent Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO: add a test suite that wraps up all these variables in a class
var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)

func init() {
	testAccProvider = New(testVersion, "")()
	testAccProviders = map[string]*schema.Provider{
		"confluent": testAccProvider,
	}
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		"confluent": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
	// Set fake values for secrets since those are required (only if not already set)
	if os.Getenv("CONFLUENT_CLOUD_API_KEY") == "" {
		_ = os.Setenv("CONFLUENT_CLOUD_API_KEY", "foo")
	}
	if os.Getenv("CONFLUENT_CLOUD_API_SECRET") == "" {
		_ = os.Setenv("CONFLUENT_CLOUD_API_SECRET", "bar")
	}
}

func TestProvider_InternalValidate(t *testing.T) {
	if err := New(testVersion, "")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestBuildUserAgent(t *testing.T) {
	// p.UserAgent appends the TF_APPEND_USER_AGENT env var if set, which would leak into the
	// assertions below, so make sure it is unset for this test.
	t.Setenv("TF_APPEND_USER_AGENT", "")

	p := New(testVersion, "")()

	base := buildUserAgent(p, testVersion, "", "")
	if !strings.Contains(base, terraformProviderUserAgent) {
		t.Fatalf("expected base user agent to contain %q, got %q", terraformProviderUserAgent, base)
	}

	tests := []struct {
		name                string
		additionalUserAgent string
		appendUserAgent     string
		wantPrefix          string
		wantSuffix          string
	}{
		{
			name:            "append_user_agent is added as the last token",
			appendUserAgent: "confluent_cloud_export",
			wantSuffix:      "confluent_cloud_export",
		},
		{
			name:            "append_user_agent is trimmed",
			appendUserAgent: "  confluent_cloud_export  ",
			wantSuffix:      "confluent_cloud_export",
		},
		{
			name:            "blank append_user_agent leaves the user agent unchanged",
			appendUserAgent: "   ",
			wantSuffix:      base,
		},
		{
			name:                "additional user agent is prepended",
			additionalUserAgent: "confluent-cli/1.2.3",
			appendUserAgent:     "confluent_cloud_export",
			wantPrefix:          "confluent-cli/1.2.3",
			wantSuffix:          "confluent_cloud_export",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildUserAgent(p, testVersion, tc.additionalUserAgent, tc.appendUserAgent)
			if tc.wantPrefix != "" && !strings.HasPrefix(got, tc.wantPrefix) {
				t.Errorf("expected user agent %q to start with %q", got, tc.wantPrefix)
			}
			if tc.wantSuffix != "" && !strings.HasSuffix(got, tc.wantSuffix) {
				t.Errorf("expected user agent %q to end with %q", got, tc.wantSuffix)
			}
		})
	}
}

func testAccPreCheck(t *testing.T) {
	ccApiKey := getEnv("CONFLUENT_CLOUD_API_KEY", "")
	ccApiSecret := getEnv("CONFLUENT_CLOUD_API_SECRET", "")
	canUseApiKeyAndSecret := ccApiKey != "" && ccApiSecret != ""
	if !canUseApiKeyAndSecret {
		t.Fatal("Both CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for acceptance tests (having them set to fake values is fine)")
	}
}

func TestProviderGoContainsTfgenMarkers(t *testing.T) {
	// Locate provider.go relative to this test file.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: unable to determine test file path")
	}
	providerGoPath := filepath.Join(filepath.Dir(thisFile), "provider.go")

	content, err := os.ReadFile(providerGoPath)
	if err != nil {
		t.Fatalf("failed to read provider.go: %s", err)
	}
	text := string(content)

	markers := []string{
		"// cli-tfgen:tf-resources",
		"// cli-tfgen:tf-datasources",
		"// cli-tfgen:tf-client-fields",
		"// cli-tfgen:tf-imports",
		"// cli-tfgen:tf-client-cfg",
		"// cli-tfgen:tf-client-endpoint",
		"// cli-tfgen:tf-client-useragent",
		"// cli-tfgen:tf-client-httpclient",
		"// cli-tfgen:tf-client-literal",
	}
	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			t.Errorf("provider.go is missing required marker %q (needed by cli-terraform-generator --provider-dir)", marker)
		}
	}
}

// TestEverySDKClientConfigGetsARetryableHTTPClient asserts that every SDK client configuration
// built in providerConfigure is also given a retryable HTTP client.
//
// providerConfigure declares one `<name>Cfg := <sdk>.NewConfiguration()` per Confluent API, and
// then assigns each one a retryable client from a separate block further down. Nothing ties the
// two lists together, so a config can be added without its retryable client and every existing
// test still passes — which is how networkingAccessPointV1Cfg ended up as the only one of 32
// without one, leaving confluent_access_point and confluent_dns_record as the only resources that
// failed immediately on a transient 429 or 5xx instead of retrying.
//
// factory_utils_test.go already covers whether the retryable client itself works. This covers
// whether every API is actually given one.
func TestEverySDKClientConfigGetsARetryableHTTPClient(t *testing.T) {
	const (
		configConstructor = "NewConfiguration"
		httpClientField   = "HTTPClient"
		retryableFactory  = "NewRetryableClientFactory"
	)

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed: unable to determine test file path")
	}
	providerGoPath := filepath.Join(filepath.Dir(thisFile), "provider.go")

	fileSet := token.NewFileSet()
	parsed, err := parser.ParseFile(fileSet, providerGoPath, nil, 0)
	if err != nil {
		t.Fatalf("failed to parse provider.go: %s", err)
	}

	var providerConfigureBody *ast.BlockStmt
	for _, decl := range parsed.Decls {
		if fn, isFunc := decl.(*ast.FuncDecl); isFunc && fn.Name.Name == "providerConfigure" {
			providerConfigureBody = fn.Body
			break
		}
	}
	if providerConfigureBody == nil {
		t.Fatal("provider.go no longer declares providerConfigure; this test needs updating")
	}

	// declared preserves source order so the failure message points at the offending line.
	var declared []string
	declaredAt := make(map[string]token.Pos)
	retryable := make(map[string]bool)

	ast.Inspect(providerConfigureBody, func(node ast.Node) bool {
		assign, isAssign := node.(*ast.AssignStmt)
		if !isAssign || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
			return true
		}

		switch assign.Tok {
		case token.DEFINE:
			// <name>Cfg := <sdk package>.NewConfiguration()
			name, isIdent := assign.Lhs[0].(*ast.Ident)
			if !isIdent {
				return true
			}
			call, isCall := assign.Rhs[0].(*ast.CallExpr)
			if !isCall {
				return true
			}
			if selector, isSelector := call.Fun.(*ast.SelectorExpr); isSelector && selector.Sel.Name == configConstructor {
				declared = append(declared, name.Name)
				declaredAt[name.Name] = name.Pos()
			}
		case token.ASSIGN:
			// <name>Cfg.HTTPClient = NewRetryableClientFactory(...).CreateRetryableClient()
			selector, isSelector := assign.Lhs[0].(*ast.SelectorExpr)
			if !isSelector || selector.Sel.Name != httpClientField {
				return true
			}
			name, isIdent := selector.X.(*ast.Ident)
			if !isIdent {
				return true
			}
			// Require the retryable factory specifically — assigning some other client would
			// otherwise satisfy this test while reintroducing the bug.
			ast.Inspect(assign.Rhs[0], func(rhs ast.Node) bool {
				if ident, isIdent := rhs.(*ast.Ident); isIdent && ident.Name == retryableFactory {
					retryable[name.Name] = true
					return false
				}
				return true
			})
		}
		return true
	})

	if len(declared) == 0 {
		t.Fatalf("found no `<name>Cfg := <sdk>.%s()` declarations in providerConfigure; this test needs updating", configConstructor)
	}

	for _, name := range declared {
		if !retryable[name] {
			t.Errorf("%s: %s is never assigned a %s.\n"+
				"Every SDK client configuration needs one, or requests to that API fail immediately on a "+
				"transient 429 or 5xx instead of retrying. Add:\n"+
				"\t%s.%s = %s(ctx, WithMaxRetries(maxRetries)).CreateRetryableClient()",
				fileSet.Position(declaredAt[name]), name, retryableFactory, name, httpClientField, retryableFactory)
		}
	}
	t.Logf("checked %d SDK client configurations", len(declared))
}

func TestSleepIfNotTestMode(t *testing.T) {
	t.Run("should not sleep in acceptance test mode (mock tests)", func(t *testing.T) {
		start := time.Now()
		SleepIfNotTestMode(time.Second, true, false)
		duration := time.Since(start)

		if duration >= time.Second {
			t.Errorf("expected no sleep, but slept for %v\n", duration)
		}
	})

	t.Run("should sleep in normal mode", func(t *testing.T) {
		start := time.Now()
		SleepIfNotTestMode(time.Second, false, false)
		duration := time.Since(start)

		if duration < time.Second {
			t.Errorf("expected to sleep, but slept for %v\n", duration)
		}
	})

	t.Run("should sleep in live production test mode", func(t *testing.T) {
		start := time.Now()
		SleepIfNotTestMode(time.Second, true, true)
		duration := time.Since(start)

		if duration < time.Second {
			t.Errorf("expected to sleep in live production test mode, but slept for %v\n", duration)
		}
	})
}

func TestGetDelayAndPollInterval(t *testing.T) {
	t.Run("should return 1s/1s in acceptance test mode", func(t *testing.T) {
		delay, pollInterval := getDelayAndPollInterval(5*time.Minute, 30*time.Second, true)

		if delay != 1*time.Second {
			t.Errorf("expected delay to be 1s in acceptance test mode, got %v", delay)
		}
		if pollInterval != 1*time.Second {
			t.Errorf("expected pollInterval to be 1s in acceptance test mode, got %v", pollInterval)
		}
	})

	t.Run("should return normal values when not in acceptance test mode", func(t *testing.T) {
		expectedDelay := 5 * time.Minute
		expectedPollInterval := 30 * time.Second
		delay, pollInterval := getDelayAndPollInterval(expectedDelay, expectedPollInterval, false)

		if delay != expectedDelay {
			t.Errorf("expected delay to be %v, got %v", expectedDelay, delay)
		}
		if pollInterval != expectedPollInterval {
			t.Errorf("expected pollInterval to be %v, got %v", expectedPollInterval, pollInterval)
		}
	})

	t.Run("should return large delay and poll interval values when not in acceptance test mode", func(t *testing.T) {
		expectedDelay := 10 * time.Minute
		expectedPollInterval := 2 * time.Minute
		delay, pollInterval := getDelayAndPollInterval(expectedDelay, expectedPollInterval, false)

		if delay != expectedDelay {
			t.Errorf("expected delay to be %v, got %v", expectedDelay, delay)
		}
		if pollInterval != expectedPollInterval {
			t.Errorf("expected pollInterval to be %v, got %v", expectedPollInterval, pollInterval)
		}
	})

	t.Run("should ignore normal values and return 1s/1s in acceptance test mode regardless of input", func(t *testing.T) {
		delay, pollInterval := getDelayAndPollInterval(10*time.Minute, 2*time.Minute, true)

		if delay != 1*time.Second {
			t.Errorf("expected delay to be 1s in acceptance test mode, got %v", delay)
		}
		if pollInterval != 1*time.Second {
			t.Errorf("expected pollInterval to be 1s in acceptance test mode, got %v", pollInterval)
		}
	})

	t.Run("should return small delay and poll interval values when not in acceptance test mode", func(t *testing.T) {
		expectedDelay := 1 * time.Second
		expectedPollInterval := 500 * time.Millisecond
		delay, pollInterval := getDelayAndPollInterval(expectedDelay, expectedPollInterval, false)

		if delay != expectedDelay {
			t.Errorf("expected delay to be %v, got %v", expectedDelay, delay)
		}
		if pollInterval != expectedPollInterval {
			t.Errorf("expected pollInterval to be %v, got %v", expectedPollInterval, pollInterval)
		}
	})
}
