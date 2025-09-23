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

func providerIntegrationV2AuthorizationDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: providerIntegrationV2AuthorizationDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the provider integration authorization.",
			},
			paramProviderIntegrationIdAuth: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the provider integration.",
			},
			paramAzureAuth: azureAuthDataSourceConfigSchema(),
			paramGcpAuth:   gcpAuthDataSourceConfigSchema(),
			paramEnvironment: environmentDataSourceSchema(),
		},
	}
}

func azureAuthDataSourceConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "Azure-specific configuration for the provider integration authorization.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			paramAzureCustomerTenantId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Customer's Azure Tenant ID.",
			},
			paramAzureConfluentMultiTenantAppId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Confluent Multi-Tenant App ID used to access customer Azure resources.",
			},
		}},
	}
}

func gcpAuthDataSourceConfigSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: "GCP-specific configuration for the provider integration authorization.",
		Elem: &schema.Resource{Schema: map[string]*schema.Schema{
			paramGcpCustomerServiceAccount: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Customer's Google Service Account that Confluent Cloud impersonates.",
			},
			paramGcpGoogleServiceAccount: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Google Service Account that Confluent Cloud uses for impersonation.",
			},
		}},
	}
}

func providerIntegrationV2AuthorizationDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	integrationId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading provider integration v2 authorization data source %q", integrationId), map[string]interface{}{providerIntegrationLoggingKey: integrationId})

	c := meta.(*Client)
	req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), integrationId).Environment(environmentId)
	integration, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error reading provider integration v2 authorization %q: %s", integrationId, createDescriptiveError(err))
	}
	integrationJson, err := json.Marshal(integration)
	if err != nil {
		return diag.Errorf("error reading provider integration v2 authorization %q: error marshaling %#v to json: %s", integrationId, integration, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched provider integration v2 authorization %q: %s", integrationId, integrationJson), map[string]interface{}{providerIntegrationLoggingKey: integrationId})

	if _, err := setProviderIntegrationV2AuthorizationAttributes(d, integration); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	return nil
}

func setProviderIntegrationV2AuthorizationAttributes(d *schema.ResourceData, integration piv2.PimV2Integration) (*schema.ResourceData, error) {
	if err := d.Set(paramProviderIntegrationIdAuth, integration.GetId()); err != nil {
		return nil, err
	}

	// Set config based on provider type
	if integration.Config != nil {
		if integration.Config.PimV2AzureIntegrationConfig != nil {
			azureConfig := integration.Config.PimV2AzureIntegrationConfig
			if err := d.Set(paramAzureAuth, []interface{}{map[string]interface{}{
				paramAzureCustomerTenantId:          azureConfig.GetCustomerAzureTenantId(),
				paramAzureConfluentMultiTenantAppId: azureConfig.GetConfluentMultiTenantAppId(),
			}}); err != nil {
				return nil, err
			}
		}

		if integration.Config.PimV2GcpIntegrationConfig != nil {
			gcpConfig := integration.Config.PimV2GcpIntegrationConfig
			if err := d.Set(paramGcpAuth, []interface{}{map[string]interface{}{
				paramGcpCustomerServiceAccount: gcpConfig.GetCustomerGoogleServiceAccount(),
				paramGcpGoogleServiceAccount:   gcpConfig.GetGoogleServiceAccount(),
			}}); err != nil {
				return nil, err
			}
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, integration.Environment.GetId(), d); err != nil {
		return nil, err
	}

	d.SetId(integration.GetId())
	return d, nil
}