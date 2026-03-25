// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// TestSDKNamingStandardCompliance verifies all SDK import aliases follow the
// naming standard: alias = removeHyphens(module directory name).
// This ensures the cli-terraform-generator can compute the correct alias
// from the SDK module name alone, without a mapping file.
func TestSDKNamingStandardCompliance(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	dir := filepath.Dir(thisFile)

	providerCode, err := os.ReadFile(filepath.Join(dir, "provider.go"))
	if err != nil {
		t.Fatalf("failed to read provider.go: %v", err)
	}
	text := string(providerCode)

	// Match import declarations: alias "github.com/confluentinc/ccloud-sdk-go-v2/MODULE/VERSION"
	importRe := regexp.MustCompile(`(\w+)\s+"github\.com/confluentinc/ccloud-sdk-go-v2/([^"]+)"`)
	matches := importRe.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		t.Fatal("no SDK imports found in provider.go")
	}

	for _, m := range matches {
		alias := m[1]
		fullPath := m[2] // e.g., "networking-access-point/v1"

		// Standard: alias = removeHyphens(fullPath) with slashes removed
		// e.g., "networking-access-point/v1" → "networkingaccesspointv1"
		expectedAlias := strings.ReplaceAll(fullPath, "-", "")
		expectedAlias = strings.ReplaceAll(expectedAlias, "/", "")

		if alias != expectedAlias {
			t.Errorf("import alias for module %q: got %q, expected %q (rule: removeHyphens(module)+version)",
				fullPath, alias, expectedAlias)
		}
	}
}
