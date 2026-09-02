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

// sdkPoster is the production Poster: it maps a Usage onto the generated
// terraform-usage/v1 contract type and POSTs it to
// <endpoint>/terraform-usage/v1/usages via the generated client (TFCA-A2).
//
// Authentication is applied per request by authFunc, not baked into the client,
// so the caller decides which identity to use. TFCA-B7 supplies a function that
// scopes reporting to the top-level Cloud API key / OAuth identity only; a nil
// authFunc sends unauthenticated (used only by tests against a local stub).
type sdkPoster struct {
	client   *terraformusagev1.APIClient
	authFunc func(context.Context) context.Context
}

// NewSDKPoster builds a Poster backed by the generated terraform-usage/v1
// client. basePath is the API origin (e.g. https://api.confluent.cloud);
// httpClient carries any transport-level configuration (a nil httpClient uses
// the SDK default). authFunc decorates each request context with credentials
// and may be nil.
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

// Post delivers one Usage. It honors ctx (the transport's per-report deadline),
// applies authentication, and returns an error on any transport failure or
// non-2xx response so the transport can log-and-drop.
func (p *sdkPoster) Post(ctx context.Context, u Usage) error {
	if p.authFunc != nil {
		ctx = p.authFunc(ctx)
	}

	resp, err := p.client.UsagesTerraformUsageV1Api.
		CreateTerraformUsageV1Usage(ctx).
		TerraformUsageV1Usage(toContractUsage(u)).
		Execute()
	if resp != nil && resp.Body != nil {
		// The generated client already drains and closes the underlying response
		// body (it returns a NopCloser over an in-memory buffer), so this Close is
		// a harmless safeguard that stays correct if that behavior ever changes.
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

// toContractUsage maps the provider-internal Usage onto the generated contract
// type. Sequence and DurationMs are int64 internally but int32 on the wire; a
// single provider process cannot approach the int32 ceiling (2.1B events, or an
// operation lasting ~24 days), so the narrowing conversion is safe in practice.
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
	// ChangedAttributes is required by the contract and non-nullable, so it must
	// serialize as [] and never null. The wrapper already guarantees a non-nil
	// slice; coerce here too so this last step before the wire enforces the
	// contract on its own rather than trusting that upstream invariant.
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
