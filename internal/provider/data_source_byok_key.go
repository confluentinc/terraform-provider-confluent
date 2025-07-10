// Copyright 2023 Confluent Inc. All Rights Reserved.
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
)

func byokDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: byokDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Description: "The id of the BYOK key",
				Required:    true,
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "A human-readable name for the BYOK key.",
				Computed:    true,
			},
			paramAws:   awsKeyDataSourceSchema(),
			paramAzure: azureKeyDataSourceSchema(),
			paramGcp:   gcpKeyDataSourceSchema(),
		},
	}
}

func awsKeyDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAwsKeyArn: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramAwsRoles: {
					Type:     schema.TypeSet,
					Elem:     &schema.Schema{Type: schema.TypeString},
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func gcpKeyDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramGcpKeyId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramGcpSecurityGroup: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func azureKeyDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramAzureKeyId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramAzureKeyVaultId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramAzureTenantId: {
					Type:     schema.TypeString,
					Computed: true,
				},
				paramAzureApplicationId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
		Computed: true,
	}
}

func byokDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	keyId := d.Get(paramId).(string)
	if keyId == "" {
		return diag.Errorf("error reading byok key: byok key id is missing")
	}

	tflog.Debug(ctx, fmt.Sprintf("Reading byok key %q=%q", paramId, keyId), map[string]interface{}{byokKeyLoggingKey: keyId})

	c := meta.(*Client)
	key, resp, err := executeKeyRead(c.byokApiContext(ctx), c, keyId)
	if err != nil {
		return diag.Errorf("error reading byok key %q: %s", keyId, createDescriptiveError(err, resp))
	}
	keyJson, err := json.Marshal(key)
	if err != nil {
		return diag.Errorf("error reading byok key %q: error marshaling %#v to json: %s", keyId, key, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched byok key %q: %s", keyId, keyJson), map[string]interface{}{byokKeyLoggingKey: keyId})

	if _, err := setKeyAttributes(d, key); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
