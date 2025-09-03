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

const (
	paramVpcEndpointServiceName                 = "vpc_endpoint_service_name"
	stateWaitingForConnections                  = "WAITING_FOR_CONNECTIONS"
	paramZone                                   = "zone"
	paramPrivateLinkServiceAlias                = "private_link_service_alias"
	paramPrivateLinkServiceResourceId           = "private_link_service_resource_id"
	paramPrivateServiceConnectServiceAttachment = "private_service_connect_service_attachment"
)

func privateLinkAttachmentDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: privateLinkAttachmentDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Private Link Attachment, for example, `platt-61ovvd`.",
			},
			paramEnvironment: environmentDataSourceSchema(),
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The display name of the Private Link Attachment.",
			},
			paramCloud: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramRegion: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramDnsDomain: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The root DNS domain for the private link attachment.",
			},
			paramAws:   awsVpcEndpointServiceSchema(),
			paramAzure: azurePrivateLinkServicesSchema(),
			paramGcp:   gcpServiceAttachmentsSchema(),
		},
	}
}

func awsVpcEndpointServiceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpcEndpointServiceName: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Id of the VPC Endpoint service.",
				},
			},
		},
	}
}

func azurePrivateLinkServicesSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateLinkServiceAlias: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Azure PrivateLink service alias for the availability zone.",
				},
				paramPrivateLinkServiceResourceId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Azure PrivateLink service resource id for the availability zone.",
				},
			},
		},
	}
}

func gcpServiceAttachmentsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateServiceConnectServiceAttachment: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "Id of a Private Service Connect Service Attachment in Confluent Cloud.",
				},
			},
		},
	}
}

func privateLinkAttachmentDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattId := d.Get(paramId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Attachment %q=%q", paramId, plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachment(c.netPLApiContext(ctx), plattId).Environment(environmentId)
	platt, resp, err := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentExecute(request)
	if err != nil {
		return diag.Errorf("error reading Private Link Attachment %q: %s", plattId, createDescriptiveError(err, resp))
	}
	plattJson, err := json.Marshal(platt)
	if err != nil {
		return diag.Errorf("error reading Private Link Attachment %q: error marshaling %#v to json: %s", plattId, platt, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Attachment %q: %s", plattId, plattJson), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	if _, err := setPrivateLinkAttachmentAttributes(d, platt); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
