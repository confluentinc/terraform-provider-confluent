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
	"fmt"
	"net/http"

	terraformusagev1 "github.com/confluentinc/ccloud-sdk-go-v2/terraform-usage/v1"
)

// sdkPoster maps a Usage onto the terraform-usage/v1 contract and POSTs it to
// <endpoint>/terraform-usage/v1/usages. authFunc applies per-request auth (a nil
// authFunc sends unauthenticated).
type sdkPoster struct {
	client   *terraformusagev1.APIClient
	authFunc func(context.Context) context.Context
}

// NewSDKPoster builds a Poster backed by the terraform-usage/v1 client. basePath
// is the API origin; a nil httpClient uses the SDK default; authFunc may be nil.
func NewSDKPoster(basePath string, httpClient *http.Client, userAgent string, authFunc func(context.Context) context.Context) Poster {
	cfg := terraformusagev1.NewConfiguration()
	if basePath != "" {
		cfg.Servers = terraformusagev1.ServerConfigurations{
			{URL: basePath},
		}
	}
	if httpClient != nil {
		cfg.HTTPClient = httpClient
	}
	if userAgent != "" {
		cfg.UserAgent = userAgent
	}
	return &sdkPoster{
		client:   terraformusagev1.NewAPIClient(cfg),
		authFunc: authFunc,
	}
}

// Post delivers one Usage: it applies auth, honors ctx, and returns an error on
// any transport failure or non-2xx response.
func (p *sdkPoster) Post(ctx context.Context, u Usage) error {
	if p.authFunc != nil {
		ctx = p.authFunc(ctx)
	}

	resp, err := p.client.UsagesTerraformUsageV1Api.
		CreateTerraformUsageV1Usage(ctx).
		TerraformUsageV1Usage(toContractUsage(u)).
		Execute()
	if resp != nil && resp.Body != nil {
		// The generated client already closes the body; this is a harmless safeguard.
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}
	if resp != nil && resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("terraform-usage endpoint returned %s", resp.Status)
	}
	return nil
}

// toContractUsage maps the internal Usage onto the wire contract. Sequence and
// DurationMs narrow int64->int32; one process cannot reach the int32 ceiling.
func toContractUsage(u Usage) terraformusagev1.TerraformUsageV1Usage {
	m := terraformusagev1.NewTerraformUsageV1Usage()
	m.SetRunId(u.RunID)
	m.SetSequence(int32(u.Sequence))
	m.SetStartedAt(u.StartedAt)
	m.SetDurationMs(int32(u.DurationMs))
	m.SetOs(u.OS)
	m.SetArch(u.Arch)
	m.SetProviderVersion(u.ProviderVersion)
	m.SetTerraformVersion(u.TerraformVersion)
	m.SetResourceType(u.ResourceType)
	m.SetOperation(string(u.Operation))
	// ChangedAttributes is required and non-nullable: coerce nil to [] so it
	// serializes as [] rather than null.
	changed := u.ChangedAttributes
	if changed == nil {
		changed = []string{}
	}
	m.SetChangedAttributes(changed)
	m.SetError(u.Error)
	if len(u.StackFrames) > 0 {
		m.SetStackFrames(u.StackFrames)
	}
	return *m
}
