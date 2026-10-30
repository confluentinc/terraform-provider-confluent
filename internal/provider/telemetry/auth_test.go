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

package telemetry

import (
	"context"
	"testing"

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"
)

func TestTokenAuthContext(t *testing.T) {
	ctx := TokenAuthContext(context.Background(), "tok-123")
	got, ok := ctx.Value(terraformusagev1.ContextAccessToken).(string)
	if !ok || got != "tok-123" {
		t.Errorf("ContextAccessToken = %q (ok=%v), want %q", got, ok, "tok-123")
	}
	// A bearer token must not also set basic-auth credentials.
	if ctx.Value(terraformusagev1.ContextBasicAuth) != nil {
		t.Errorf("TokenAuthContext must not set ContextBasicAuth")
	}
}

func TestBasicAuthContext(t *testing.T) {
	ctx := BasicAuthContext(context.Background(), "key", "secret")
	got, ok := ctx.Value(terraformusagev1.ContextBasicAuth).(terraformusagev1.BasicAuth)
	if !ok {
		t.Fatalf("ContextBasicAuth not set or wrong type: %T", ctx.Value(terraformusagev1.ContextBasicAuth))
	}
	if got.UserName != "key" || got.Password != "secret" {
		t.Errorf("BasicAuth = %+v, want {UserName:key Password:secret}", got)
	}
	// Basic auth must not also set a bearer token.
	if ctx.Value(terraformusagev1.ContextAccessToken) != nil {
		t.Errorf("BasicAuthContext must not set ContextAccessToken")
	}
}
