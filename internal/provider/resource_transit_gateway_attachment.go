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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"regexp"
	"strings"
)

const (
	paramRamResourceShareArn        = "ram_resource_share_arn"
	paramTransitGatewayId           = "transit_gateway_id"
	paramEnableCustomRoutes         = "enable_custom_routes"
	paramTransitGatewayAttachmentId = "transit_gateway_attachment_id"
	awsTransitGatewayAttachmentKind = "AwsTransitGatewayAttachment"
)

var paramAwsRamResourceShareArn = fmt.Sprintf("%s.0.%s", paramAws, paramRamResourceShareArn)
var paramAwsTransitGatewayId = fmt.Sprintf("%s.0.%s", paramAws, paramTransitGatewayId)
var paramAwsEnableCustomRoutes = fmt.Sprintf("%s.0.%s", paramAws, paramEnableCustomRoutes)

func transitGatewayAttachmentResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: transitGatewayAttachmentCreate,
		ReadContext:   transitGatewayAttachmentRead,
		UpdateContext: transitGatewayAttachmentUpdate,
		DeleteContext: transitGatewayAttachmentDelete,
		Importer: &schema.ResourceImporter{
			StateContext: transitGatewayAttachmentImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:        schema.TypeString,
				Description: "The name of the Transit Gateway Attachment.",
				Optional:    true,
				Computed:    true,
			},
			paramAws:         awsTransitGatewayAttachmentSchema(),
			paramNetwork:     requiredNetworkSchema(),
			paramEnvironment: environmentSchema(),
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(networkingAPICreateTimeout),
			Delete: schema.DefaultTimeout(networkingAPIDeleteTimeout),
		},
	}
}

func transitGatewayAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	isAwsTransitGatewayAttachment := len(d.Get(paramAws).([]interface{})) > 0

	spec := net.NewNetworkingV1TransitGatewayAttachmentSpec()
	if displayName != "" {
		spec.SetDisplayName(displayName)
	}
	if isAwsTransitGatewayAttachment {
		ramResourceShareArn := d.Get(paramAwsRamResourceShareArn).(string)
		transitGatewayId := d.Get(paramAwsTransitGatewayId).(string)
		awsTransitGatewayAttachment := net.NewNetworkingV1AwsTransitGatewayAttachment(awsTransitGatewayAttachmentKind, ramResourceShareArn, transitGatewayId)
		isRoutesAttributeSet := len(d.Get(paramAwsRoutes).([]interface{})) > 0
		if isRoutesAttributeSet {
			routes := convertToStringSlice(d.Get(paramAwsRoutes).([]interface{}))
			awsTransitGatewayAttachment.SetRoutes(routes)
		}
		enableCustomerRoutes := d.Get(paramAwsEnableCustomRoutes).(bool)
		awsTransitGatewayAttachment.SetEnableCustomRoutes(enableCustomerRoutes)
		spec.SetCloud(net.NetworkingV1TransitGatewayAttachmentSpecCloudOneOf{NetworkingV1AwsTransitGatewayAttachment: awsTransitGatewayAttachment})
	} else {
		return diag.Errorf("None of %q blocks was provided for confluent_transit_gateway_attachment resource", paramAws)
	}
	spec.SetNetwork(net.ObjectReference{Id: networkId})
	spec.SetEnvironment(net.ObjectReference{Id: environmentId})

	createTransitGatewayAttachmentRequest := net.NetworkingV1TransitGatewayAttachment{Spec: spec}
	createTransitGatewayAttachmentRequestJson, err := json.Marshal(createTransitGatewayAttachmentRequest)
	if err != nil {
		return diag.Errorf("error creating Transit Gateway Attachment: error marshaling %#v to json: %s", createTransitGatewayAttachmentRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Transit Gateway Attachment: %s", createTransitGatewayAttachmentRequestJson))

	createdTransitGatewayAttachment, _, err := executeTransitGatewayAttachmentCreate(c.netApiContext(ctx), c, createTransitGatewayAttachmentRequest)
	if err != nil {
		return diag.Errorf("error creating Transit Gateway Attachment %q: %s", createdTransitGatewayAttachment.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdTransitGatewayAttachment.GetId())

	if err := waitForTransitGatewayAttachmentToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Transit Gateway Attachment %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdTransitGatewayAttachmentJson, err := json.Marshal(createdTransitGatewayAttachment)
	if err != nil {
		return diag.Errorf("error creating Transit Gateway Attachment %q: error marshaling %#v to json: %s", d.Id(), createdTransitGatewayAttachment, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Transit Gateway Attachment %q: %s", d.Id(), createdTransitGatewayAttachmentJson), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	return transitGatewayAttachmentRead(ctx, d, meta)
}

func executeTransitGatewayAttachmentCreate(ctx context.Context, c *Client, transitGatewayAttachment net.NetworkingV1TransitGatewayAttachment) (net.NetworkingV1TransitGatewayAttachment, *http.Response, error) {
	req := c.netClient.TransitGatewayAttachmentsNetworkingV1Api.CreateNetworkingV1TransitGatewayAttachment(c.netApiContext(ctx)).NetworkingV1TransitGatewayAttachment(transitGatewayAttachment)
	return req.Execute()
}

func executeTransitGatewayAttachmentRead(ctx context.Context, c *Client, environmentId string, transitGatewayAttachmentId string) (net.NetworkingV1TransitGatewayAttachment, *http.Response, error) {
	req := c.netClient.TransitGatewayAttachmentsNetworkingV1Api.GetNetworkingV1TransitGatewayAttachment(c.netApiContext(ctx), transitGatewayAttachmentId).Environment(environmentId)
	return req.Execute()
}

func transitGatewayAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	transitGatewayAttachmentId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readTransitGatewayAttachmentAndSetAttributes(ctx, d, meta, environmentId, transitGatewayAttachmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Transit Gateway Attachment %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readTransitGatewayAttachmentAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, transitGatewayAttachmentId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	transitGatewayAttachment, resp, err := executeTransitGatewayAttachmentRead(c.netApiContext(ctx), c, environmentId, transitGatewayAttachmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Transit Gateway Attachment %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Transit Gateway Attachment %q in TF state because Transit Gateway Attachment could not be found on the server", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	transitGatewayAttachmentJson, err := json.Marshal(transitGatewayAttachment)
	if err != nil {
		return nil, fmt.Errorf("error reading Transit Gateway Attachment %q: error marshaling %#v to json: %s", transitGatewayAttachmentId, transitGatewayAttachment, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Transit Gateway Attachment %q: %s", d.Id(), transitGatewayAttachmentJson), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	if _, err := setTransitGatewayAttachmentAttributes(d, transitGatewayAttachment); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setTransitGatewayAttachmentAttributes(d *schema.ResourceData, transitGatewayAttachment net.NetworkingV1TransitGatewayAttachment) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, transitGatewayAttachment.Spec.GetDisplayName()); err != nil {
		return nil, err
	}

	if transitGatewayAttachment.Spec.Cloud.NetworkingV1AwsTransitGatewayAttachment != nil {
		if err := d.Set(paramAws, []interface{}{map[string]interface{}{
			paramRamResourceShareArn:        transitGatewayAttachment.Spec.Cloud.NetworkingV1AwsTransitGatewayAttachment.GetRamShareArn(),
			paramTransitGatewayId:           transitGatewayAttachment.Spec.Cloud.NetworkingV1AwsTransitGatewayAttachment.GetTransitGatewayId(),
			paramRoutes:                     transitGatewayAttachment.Spec.Cloud.NetworkingV1AwsTransitGatewayAttachment.GetRoutes(),
			paramEnableCustomRoutes:         transitGatewayAttachment.Spec.Cloud.NetworkingV1AwsTransitGatewayAttachment.GetEnableCustomRoutes(),
			paramTransitGatewayAttachmentId: transitGatewayAttachment.Status.Cloud.NetworkingV1AwsTransitGatewayAttachmentStatus.GetTransitGatewayAttachmentId(),
		}}); err != nil {
			return nil, err
		}
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, transitGatewayAttachment.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, transitGatewayAttachment.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	d.SetId(transitGatewayAttachment.GetId())
	return d, nil
}

func transitGatewayAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	c := meta.(*Client)

	req := c.netClient.TransitGatewayAttachmentsNetworkingV1Api.DeleteNetworkingV1TransitGatewayAttachment(c.netApiContext(ctx), d.Id()).Environment(environmentId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Transit Gateway Attachment %q: %s", d.Id(), createDescriptiveError(err))
	}

	if err := waitForTransitGatewayAttachmentToBeDeleted(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Transit Gateway Attachment %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	return nil
}

func transitGatewayAttachmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Transit Gateway Attachment %q: only %q attribute can be updated for Transit Gateway Attachment", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	updatedDisplayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	updateTransitGatewayAttachmentRequest := net.NewNetworkingV1TransitGatewayAttachmentUpdate()
	updateSpec := net.NewNetworkingV1TransitGatewayAttachmentSpecUpdate()
	updateSpec.SetDisplayName(updatedDisplayName)
	updateSpec.SetEnvironment(net.ObjectReference{Id: environmentId})
	updateTransitGatewayAttachmentRequest.SetSpec(*updateSpec)
	updateTransitGatewayAttachmentRequestJson, err := json.Marshal(updateTransitGatewayAttachmentRequest)
	if err != nil {
		return diag.Errorf("error updating Transit Gateway Attachment %q: error marshaling %#v to json: %s", d.Id(), updateTransitGatewayAttachmentRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Transit Gateway Attachment %q: %s", d.Id(), updateTransitGatewayAttachmentRequestJson), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	req := c.netClient.TransitGatewayAttachmentsNetworkingV1Api.UpdateNetworkingV1TransitGatewayAttachment(c.netApiContext(ctx), d.Id()).NetworkingV1TransitGatewayAttachmentUpdate(*updateTransitGatewayAttachmentRequest)
	updatedTransitGatewayAttachment, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Transit Gateway Attachment %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedTransitGatewayAttachmentJson, err := json.Marshal(updatedTransitGatewayAttachment)
	if err != nil {
		return diag.Errorf("error updating Transit Gateway Attachment %q: error marshaling %#v to json: %s", d.Id(), updatedTransitGatewayAttachment, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Transit Gateway Attachment %q: %s", d.Id(), updatedTransitGatewayAttachmentJson), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})
	return transitGatewayAttachmentRead(ctx, d, meta)
}

func transitGatewayAttachmentImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})

	envIDAndTransitGatewayAttachmentId := d.Id()
	parts := strings.Split(envIDAndTransitGatewayAttachmentId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Transit Gateway Attachment: invalid format: expected '<env ID>/<tgwa ID>'")
	}

	environmentId := parts[0]
	transitGatewayAttachmentId := parts[1]
	d.SetId(transitGatewayAttachmentId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readTransitGatewayAttachmentAndSetAttributes(ctx, d, meta, environmentId, transitGatewayAttachmentId); err != nil {
		return nil, fmt.Errorf("error importing Transit Gateway Attachment %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Transit Gateway Attachment %q", d.Id()), map[string]interface{}{transitGatewayAttachmentLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func awsTransitGatewayAttachmentSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramRamResourceShareArn: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The Amazon Resource Name (ARN) of the Resource Access Manager (RAM) Resource Share of the transit gateway your Confluent Cloud network attaches to.",
					// TODO: add validation func
				},
				paramTransitGatewayId: {
					Type:         schema.TypeString,
					Required:     true,
					ForceNew:     true,
					Description:  "The ID of the AWS Transit Gateway that your Confluent Cloud network attaches to.",
					ValidateFunc: validation.StringMatch(regexp.MustCompile("^tgw-"), "AWS Transit Gateway ID must start with 'tgw-'."),
				},
				paramEnableCustomRoutes: {
					Type:        schema.TypeBool,
					Optional:    true,
					ForceNew:    true,
					Default:     false,
					Description: "Enable custom destination routes in Confluent Cloud.",
				},
				paramRoutes: {
					Type:        schema.TypeList,
					Optional:    true,
					ForceNew:    true,
					Computed:    true,
					MinItems:    1,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Description: "List of destination routes for traffic from Confluent VPC to customer VPC via Transit Gateway.",
					// TODO: add validation func
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
