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

func privateLinkAttachmentConnectionDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: privateLinkAttachmentConnectionDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Private Link Attachment Connection, for example, `plattc-61ovvd`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Private Link Attachment Connection.",
				Computed:    true,
			},
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramEnvironment:           environmentDataSourceSchema(),
			paramPrivateLinkAttachment: privateLinkAttachmentDataSourceSchema(),
			paramAws:                   awsPlattcDataSourceSchema(),
			paramAzure:                 azurePlattcDataSourceSchema(),
			paramGcp:                   gcpPlattcDataSourceSchema(),
		},
	}
}

func privateLinkAttachmentDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The unique identifier for the private link attachment.",
				},
			},
		},
		Computed:    true,
		Description: "The private_link_attachment to which this belongs.",
	}
}

func awsPlattcDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpcEndpointId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Id of a VPC Endpoint that is connected to the VPC Endpoint service.",
				},
			},
		},
	}
}

func azurePlattcDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateEndpointResourceId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Resource Id of the PrivateEndpoint that is connected to the PrivateLink service.",
				},
			},
		},
	}
}

func gcpPlattcDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateServiceConnectConnectionId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Id of the Private Service connection.",
				},
			},
		},
	}
}

func privateLinkAttachmentConnectionDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattcId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Attachment Connection %q=%q", paramId, plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentConnection(c.netPLApiContext(ctx), plattcId).Environment(environmentId)
	plattc, _, err := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentConnectionExecute(request)
	if err != nil {
		return diag.Errorf("error reading Private Link Attachment Connection %q: %s", plattcId, createDescriptiveError(err))
	}
	plattcJson, err := json.Marshal(plattc)
	if err != nil {
		return diag.Errorf("error reading Private Link Attachment Connection %q: error marshaling %#v to json: %s", plattcId, plattc, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Attachment Connection %q: %s", plattcId, plattcJson), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	if _, err := setPrivateLinkAttachmentConnectionAttributes(d, plattc); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
