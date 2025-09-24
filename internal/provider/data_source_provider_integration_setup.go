// Copyright 2025 Confluent Inc. All Rights Reserved.
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
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	piv2 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v2"
)

func providerIntegrationSetupDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: providerIntegrationSetupDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID for provider integration setup.",
			},
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The display name of provider integration setup.",
			},
			paramCloudProvider: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cloud service provider for the integration.",
			},
			paramStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the provider integration.",
			},
			paramUsages: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The usages of provider integration.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			paramEnvironment: environmentDataSourceSchema(),
		},
	}
}

func providerIntegrationSetupDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration v2 data source"))

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	integrationId := d.Get(paramId).(string)
	if integrationId != "" {
		return providerIntegrationSetupDataSourceReadUsingId(ctx, d, meta, environmentId, integrationId)
	} else {
		displayName := d.Get(paramDisplayName).(string)
		return providerIntegrationSetupDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	}
}

func providerIntegrationSetupDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, integrationId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration v2 %q=%q", paramId, integrationId), map[string]interface{}{providerIntegrationLoggingKey: integrationId})

	c := meta.(*Client)
	req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), integrationId).Environment(environmentId)
	integration, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading provider integration v2 %q: %s", integrationId, createDescriptiveError(err))
	}
	integrationJson, err := json.Marshal(integration)
	if err != nil {
		return diag.Errorf("error reading provider integration v2 %q: error marshaling %#v to json: %s", integrationId, integration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration v2 %q: %s", integrationId, integrationJson), map[string]interface{}{providerIntegrationLoggingKey: integrationId})

	if _, err := setProviderIntegrationV2Attributes(d, integration); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func providerIntegrationSetupDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration v2 %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	req := c.piV2Client.IntegrationsPimV2Api.ListPimV2Integrations(c.piV2ApiContext(ctx)).Environment(environmentId).DisplayName(displayName)
	integrations, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading provider integration v2 %q: %s", displayName, createDescriptiveError(err))
	}
	if len(integrations.GetData()) == 0 {
		return diag.Errorf("provider integration v2 with display name %q was not found", displayName)
	}

	integration := integrations.GetData()[0]
	integrationJson, err := json.Marshal(integration)
	if err != nil {
		return diag.Errorf("error reading provider integration v2 %q: error marshaling %#v to json: %s", displayName, integration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration v2 %q: %s", displayName, integrationJson))

	if _, err := setProviderIntegrationV2Attributes(d, integration); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func setProviderIntegrationV2Attributes(d *schema.ResourceData, integration piv2.PimV2Integration) (*schema.ResourceData, error) {
	if err := d.Set(paramId, integration.GetId()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDisplayName, integration.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloudProvider, integration.GetProvider()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatus, integration.GetStatus()); err != nil {
		return nil, err
	}
	if err := d.Set(paramUsages, integration.GetUsages()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, integration.Environment.GetId(), d); err != nil {
		return nil, err
	}

	d.SetId(integration.GetId())
	return d, nil
}