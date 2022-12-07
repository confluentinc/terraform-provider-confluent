// Copyright 2022 Confluent Inc. All Rights Reserved.
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
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	// The maximum allowable page size - 1 (to avoid off-by-one errors) when listing transit gateway attachments using Networking API
	// https://docs.confluent.io/cloud/current/api.html#operation/listNetworkingV1TransitGatewayAttachments
	listTransitGatewayAttachmentsPageSize = 99
)

func transitGatewayAttachmentDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: transitGatewayAttachmentDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
				Description:  "The ID of the TransitGatewayAttachment, for example, `pla-abc123`.",
			},
			// Similarly, paramEnvironment is required as well
			paramEnvironment: environmentDataSourceSchema(),
			paramNetwork:     networkDataSourceSchema(),
			paramDisplayName: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				// A user should provide a value for either "id" or "display_name" attribute, not both
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramAws: awsTransitGatewayAttachmentDataSourceSchema(),
		},
	}
}

func transitGatewayAttachmentDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// ExactlyOneOf specified in the schema ensures one of paramId or paramDisplayName is specified.
	// The next step is to figure out which one exactly is set.
	transitGatewayAttachmentId := d.Get(paramId).(string)
	displayName := d.Get(paramDisplayName).(string)

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if transitGatewayAttachmentId != "" {
		return transitGatewayAttachmentDataSourceReadUsingId(ctx, d, meta, environmentId, transitGatewayAttachmentId)
	} else if displayName != "" {
		return transitGatewayAttachmentDataSourceReadUsingDisplayName(ctx, d, meta, environmentId, displayName)
	} else {
		return diag.Errorf("error reading Transit Gateway Attachment: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func transitGatewayAttachmentDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Transit Gateway Attachment %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	transitGatewayAttachments, err := loadTransitGatewayAttachments(ctx, c, environmentId)
	if err != nil {
		return diag.Errorf("error reading Transit Gateway Attachment %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleTransitGatewayAttachmentsWithTargetDisplayName(transitGatewayAttachments, displayName) {
		return diag.Errorf("error reading Transit Gateway Attachment: there are multiple Transit Gateway Attachment with %q=%q", paramDisplayName, displayName)
	}

	for _, transitGatewayAttachment := range transitGatewayAttachments {
		if transitGatewayAttachment.Spec.GetDisplayName() == displayName {
			if _, err := setTransitGatewayAttachmentAttributes(d, transitGatewayAttachment); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Transit Gateway Attachment: Transit Gateway Attachment with %q=%q was not found", paramDisplayName, displayName)
}

func transitGatewayAttachmentDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, transitGatewayAttachmentId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Transit Gateway Attachment %q=%q", paramId, transitGatewayAttachmentId), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})

	c := meta.(*Client)
	transitGatewayAttachment, _, err := executeTransitGatewayAttachmentRead(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId)
	if err != nil {
		return diag.Errorf("error reading Transit Gateway Attachment %q: %s", transitGatewayAttachmentId, createDescriptiveError(err))
	}
	transitGatewayAttachmentJson, err := json.Marshal(transitGatewayAttachment)
	if err != nil {
		return diag.Errorf("error reading Transit Gateway Attachment %q: error marshaling %#v to json: %s", transitGatewayAttachmentId, transitGatewayAttachment, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Transit Gateway Attachment %q: %s", transitGatewayAttachmentId, transitGatewayAttachmentJson), map[string]interface{}{transitGatewayAttachmentLoggingKey: transitGatewayAttachmentId})

	if _, err := setTransitGatewayAttachmentAttributes(d, transitGatewayAttachment); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func orgHasMultipleTransitGatewayAttachmentsWithTargetDisplayName(transitGatewayAttachments []net.NetworkingV1TransitGatewayAttachment, displayName string) bool {
	var numberOfTransitGatewayAttachmentsWithTargetDisplayName = 0
	for _, transitGatewayAttachment := range transitGatewayAttachments {
		if transitGatewayAttachment.Spec.GetDisplayName() == displayName {
			numberOfTransitGatewayAttachmentsWithTargetDisplayName += 1
		}
	}
	return numberOfTransitGatewayAttachmentsWithTargetDisplayName > 1
}

func loadTransitGatewayAttachments(ctx context.Context, c *Client, environmentId string) ([]net.NetworkingV1TransitGatewayAttachment, error) {
	transitGatewayAttachments := make([]net.NetworkingV1TransitGatewayAttachment, 0)

	allTransitGatewayAttachmentsAreCollected := false
	pageToken := ""
	for !allTransitGatewayAttachmentsAreCollected {
		transitGatewayAttachmentsPageList, _, err := executeListTransitGatewayAttachments(ctx, c, environmentId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading TransitGatewayAttachments: %s", createDescriptiveError(err))
		}
		transitGatewayAttachments = append(transitGatewayAttachments, transitGatewayAttachmentsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := transitGatewayAttachmentsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allTransitGatewayAttachmentsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading TransitGatewayAttachments: %s", createDescriptiveError(err))
				}
			}
		} else {
			allTransitGatewayAttachmentsAreCollected = true
		}
	}
	return transitGatewayAttachments, nil
}

func executeListTransitGatewayAttachments(ctx context.Context, c *Client, environmentId, pageToken string) (net.NetworkingV1TransitGatewayAttachmentList, *http.Response, error) {
	if pageToken != "" {
		return c.netClient.TransitGatewayAttachmentsNetworkingV1Api.ListNetworkingV1TransitGatewayAttachments(c.netApiContext(ctx)).Environment(environmentId).PageSize(listTransitGatewayAttachmentsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.netClient.TransitGatewayAttachmentsNetworkingV1Api.ListNetworkingV1TransitGatewayAttachments(c.netApiContext(ctx)).Environment(environmentId).PageSize(listTransitGatewayAttachmentsPageSize).Execute()
	}
}

func awsTransitGatewayAttachmentDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRamResourceShareArn: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The Amazon Resource Name (ARN) of the Resource Access Manager (RAM) Resource Share of the transit gateway your Confluent Cloud network attaches to.",
				},
				paramTransitGatewayId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the AWS Transit Gateway that your Confluent Cloud network attaches to.",
				},
				paramRoutes: {
					Type:        schema.TypeList,
					Computed:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "List of destination routes for traffic from Confluent VPC to customer VPC via Transit Gateway.",
				},
				paramTransitGatewayAttachmentId: {
					Type:        schema.TypeString,
					Computed:    true,
					Description: "The ID of the AWS Transit Gateway VPC Attachment that attaches Confluent VPC to Transit Gateway.",
				},
			},
		},
	}
}
