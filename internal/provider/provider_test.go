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
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO: add a test suite that wraps up all these variables in a class
var testAccProviders map[string]*schema.Provider
var testAccProvider *schema.Provider
var testAccProviderFactories map[string]func() (*schema.Provider, error)

const (
	testVersion = "test-version"
)

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

func testAccPreCheck(t *testing.T) {
	ccApiKey := getEnv("CONFLUENT_CLOUD_API_KEY", "")
	ccApiSecret := getEnv("CONFLUENT_CLOUD_API_SECRET", "")
	canUseApiKeyAndSecret := ccApiKey != "" && ccApiSecret != ""
	if !canUseApiKeyAndSecret {
		t.Fatal("Both CONFLUENT_CLOUD_API_KEY and CONFLUENT_CLOUD_API_SECRET must be set for acceptance tests (having them set to fake values is fine)")
	}
}

func TestSleepIfNotTestMode(t *testing.T) {
	t.Run("should not sleep in acceptance test mode", func(t *testing.T) {
		start := time.Now()
		SleepIfNotTestMode(time.Second, true)
		duration := time.Since(start)

		if duration >= time.Second {
			t.Errorf("expected no sleep, but slept for %v\n", duration)
		}
	})

	t.Run("should sleep in normal mode", func(t *testing.T) {
		start := time.Now()
		SleepIfNotTestMode(time.Second, false)
		duration := time.Since(start)

		if duration < time.Second {
			t.Errorf("expected to sleep, but slept for %v\n", duration)
		}
	})
}
