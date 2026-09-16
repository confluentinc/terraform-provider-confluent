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

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"
)

// Auth decorators for the terraform-usage client: they attach credentials to a
// request context, which the generated client reads when building the request.

// TokenAuthContext attaches an OAuth/STS bearer access token.
func TokenAuthContext(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, terraformusagev1.ContextAccessToken, accessToken)
}

// BasicAuthContext attaches a Cloud API key/secret as HTTP basic auth.
func BasicAuthContext(ctx context.Context, apiKey, apiSecret string) context.Context {
	return context.WithValue(ctx, terraformusagev1.ContextBasicAuth, terraformusagev1.BasicAuth{
		UserName: apiKey,
		Password: apiSecret,
	})
}
