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

package telemetry

import (
	"context"

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"
)

// This file keeps knowledge of the generated client's auth context keys inside
// the telemetry package, so the provider wiring can build an auth decorator from
// plain credentials without importing the SDK itself. Which credential to use is
// the caller's decision (TFCA-B7 scopes it to the top-level Cloud identity).

// BasicAuthContext returns an auth decorator that authenticates terraform-usage
// requests with a Cloud API key/secret over HTTP Basic auth.
func BasicAuthContext(key, secret string) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, terraformusagev1.ContextBasicAuth, terraformusagev1.BasicAuth{
			UserName: key,
			Password: secret,
		})
	}
}

// TokenAuthContext returns an auth decorator that authenticates with a bearer
// access token (OAuth or STS).
func TokenAuthContext(accessToken string) func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		return context.WithValue(ctx, terraformusagev1.ContextAccessToken, accessToken)
	}
}
