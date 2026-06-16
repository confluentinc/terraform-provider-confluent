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
	"testing"
)

func TestApplyOAuthDefaults_NoUserConfigs(t *testing.T) {
	// When the user provides no OAuth configs, all values should come from the provider token.
	configs := map[string]string{}
	token := &OAuthToken{
		ClientId:       "provider-client-id",
		ClientSecret:   "provider-client-secret",
		TokenUrl:       "https://provider-token-url",
		IdentityPoolId: "provider-pool",
		Scope:          "provider-scope",
	}

	applyOAuthDefaults(configs, token, "lsrc-dest")

	expectedConfigs := map[string]string{
		bearerAuthClientId:          "provider-client-id",
		bearerAuthClientSecret:      "provider-client-secret",
		bearerAuthIssuerEndpointUrl: "https://provider-token-url",
		bearerAuthCredentialsSource: configOAuthBearer,
		bearerAuthIdentityPoolId:    "provider-pool",
		bearerAuthLogicalCluster:    "lsrc-dest",
		bearerAuthScope:             "provider-scope",
	}

	if len(configs) != len(expectedConfigs) {
		t.Errorf("expected %d config keys, got %d", len(expectedConfigs), len(configs))
	}
	for key, expected := range expectedConfigs {
		if configs[key] != expected {
			t.Errorf("expected configs[%q] = %q, got %q", key, expected, configs[key])
		}
	}
}

func TestApplyOAuthDefaults_UserConfigsTakePrecedence(t *testing.T) {
	// When the user provides destination-specific OAuth configs, those should NOT be overwritten.
	configs := map[string]string{
		bearerAuthClientId:          "dest-client-id",
		bearerAuthClientSecret:      "dest-client-secret",
		bearerAuthIssuerEndpointUrl: "https://dest-token-url",
		bearerAuthCredentialsSource: configOAuthBearer,
		bearerAuthIdentityPoolId:    "dest-pool",
		bearerAuthLogicalCluster:    "lsrc-dest-user",
		bearerAuthScope:             "dest-scope",
	}
	token := &OAuthToken{
		ClientId:       "provider-client-id",
		ClientSecret:   "provider-client-secret",
		TokenUrl:       "https://provider-token-url",
		IdentityPoolId: "provider-pool",
		Scope:          "provider-scope",
	}

	applyOAuthDefaults(configs, token, "lsrc-dest-from-block")

	expectedConfigs := map[string]string{
		bearerAuthClientId:          "dest-client-id",
		bearerAuthClientSecret:      "dest-client-secret",
		bearerAuthIssuerEndpointUrl: "https://dest-token-url",
		bearerAuthCredentialsSource: configOAuthBearer,
		bearerAuthIdentityPoolId:    "dest-pool",
		bearerAuthLogicalCluster:    "lsrc-dest-user",
		bearerAuthScope:             "dest-scope",
	}

	if len(configs) != len(expectedConfigs) {
		t.Errorf("expected %d config keys, got %d", len(expectedConfigs), len(configs))
	}
	for key, expected := range expectedConfigs {
		if configs[key] != expected {
			t.Errorf("expected configs[%q] = %q, got %q", key, expected, configs[key])
		}
	}
}

func TestApplyOAuthDefaults_PartialUserConfigs(t *testing.T) {
	// When the user provides some configs, only the missing ones should be filled from the provider.
	configs := map[string]string{
		bearerAuthClientId:     "dest-client-id",
		bearerAuthClientSecret: "dest-client-secret",
	}
	token := &OAuthToken{
		ClientId:       "provider-client-id",
		ClientSecret:   "provider-client-secret",
		TokenUrl:       "https://provider-token-url",
		IdentityPoolId: "provider-pool",
		Scope:          "provider-scope",
	}

	applyOAuthDefaults(configs, token, "lsrc-dest")

	// User-specified values should be preserved
	if configs[bearerAuthClientId] != "dest-client-id" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthClientId, "dest-client-id", configs[bearerAuthClientId])
	}
	if configs[bearerAuthClientSecret] != "dest-client-secret" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthClientSecret, "dest-client-secret", configs[bearerAuthClientSecret])
	}

	// Missing values should come from the provider
	if configs[bearerAuthIssuerEndpointUrl] != "https://provider-token-url" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthIssuerEndpointUrl, "https://provider-token-url", configs[bearerAuthIssuerEndpointUrl])
	}
	if configs[bearerAuthIdentityPoolId] != "provider-pool" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthIdentityPoolId, "provider-pool", configs[bearerAuthIdentityPoolId])
	}
	if configs[bearerAuthLogicalCluster] != "lsrc-dest" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthLogicalCluster, "lsrc-dest", configs[bearerAuthLogicalCluster])
	}
	if configs[bearerAuthScope] != "provider-scope" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthScope, "provider-scope", configs[bearerAuthScope])
	}
	if configs[bearerAuthCredentialsSource] != configOAuthBearer {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthCredentialsSource, configOAuthBearer, configs[bearerAuthCredentialsSource])
	}
}

func TestApplyOAuthDefaults_EmptyProviderScope(t *testing.T) {
	// When the provider scope is empty and user doesn't set it, it should NOT be added.
	configs := map[string]string{}
	token := &OAuthToken{
		ClientId:       "provider-client-id",
		ClientSecret:   "provider-client-secret",
		TokenUrl:       "https://provider-token-url",
		IdentityPoolId: "provider-pool",
		Scope:          "",
	}

	applyOAuthDefaults(configs, token, "lsrc-dest")

	if _, ok := configs[bearerAuthScope]; ok {
		t.Errorf("expected configs[%q] to not be set when provider scope is empty, got %q", bearerAuthScope, configs[bearerAuthScope])
	}
}

func TestApplyOAuthDefaults_EmptyProviderScopeUserScopePreserved(t *testing.T) {
	// When the provider scope is empty but user sets it, the user value should be preserved.
	configs := map[string]string{
		bearerAuthScope: "user-scope",
	}
	token := &OAuthToken{
		ClientId:       "provider-client-id",
		ClientSecret:   "provider-client-secret",
		TokenUrl:       "https://provider-token-url",
		IdentityPoolId: "provider-pool",
		Scope:          "",
	}

	applyOAuthDefaults(configs, token, "lsrc-dest")

	if configs[bearerAuthScope] != "user-scope" {
		t.Errorf("expected configs[%q] = %q, got %q", bearerAuthScope, "user-scope", configs[bearerAuthScope])
	}
}
