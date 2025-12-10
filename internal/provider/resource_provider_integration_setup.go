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
	"fmt"
	"strings"

	piv2 "github.com/confluentinc/ccloud-sdk-go-v2/provider-integration/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	// Provider constants
	providerAzure = "AZURE"
	providerGcp   = "GCP"
)

// providerIntegrationSetupResource defines the setup/creation resource for PIM v2 integrations.
// This resource only handles POST (creates DRAFT integration). Use confluent_provider_integration_authorization for config/validation.
func providerIntegrationSetupResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: providerIntegrationSetupCreate,
		ReadContext:   providerIntegrationSetupRead,
		DeleteContext: providerIntegrationSetupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: providerIntegrationSetupImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID for provider integration setup.",
			},
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Display name of Provider Integration.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider in which the network exists.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
				ForceNew:     true,
			},
			paramUsages: {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of resource CRNs where this integration is used.",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			paramStatus: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the provider integration (DRAFT, CREATED, ACTIVE).",
			},

			paramEnvironment: environmentSchema(),
		},
	}
}

func providerIntegrationSetupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	provider := d.Get(paramCloud).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	createReq := piv2.PimV2Integration{}
	createReq.SetDisplayName(displayName)
	createReq.SetProvider(strings.ToLower(provider))
	createReq.SetEnvironment(piv2.ObjectReference{Id: environmentId})

	req := c.piV2Client.IntegrationsPimV2Api.CreatePimV2Integration(c.piV2ApiContext(ctx)).PimV2Integration(createReq)
	created, _, err := req.Execute()
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	d.SetId(created.GetId())
	return providerIntegrationSetupRead(ctx, d, meta)
}

func providerIntegrationSetupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	req := c.piV2Client.IntegrationsPimV2Api.GetPimV2Integration(c.piV2ApiContext(ctx), d.Id()).Environment(environmentId)
	pim, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading provider integration setup %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing provider integration setup %q in TF state because it could not be found on the server", d.Id()), map[string]interface{}{providerIntegrationLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}
		return diag.FromErr(createDescriptiveError(err))
	}

	if err := d.Set(paramDisplayName, pim.GetDisplayName()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(paramCloud, strings.ToUpper(pim.GetProvider())); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(paramStatus, pim.GetStatus()); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(paramUsages, pim.GetUsages()); err != nil {
		return diag.FromErr(err)
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, pim.Environment.GetId(), d); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(pim.GetId())
	return nil
}

func providerIntegrationSetupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	req := c.piV2Client.IntegrationsPimV2Api.DeletePimV2Integration(c.piV2ApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()
	if err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func providerIntegrationSetupImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Expect format: <env-id>/<integration-id>
	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing provider integration setup: invalid format: expected '<env ID>/<integration ID>'")
	}
	environmentId := parts[0]
	integrationId := parts[1]
	d.SetId(integrationId)
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, environmentId, d); err != nil {
		return nil, err
	}
	tflog.Debug(ctx, fmt.Sprintf("Imported provider integration setup %q (environment %q)", d.Id(), environmentId))
	return []*schema.ResourceData{d}, nil
}
