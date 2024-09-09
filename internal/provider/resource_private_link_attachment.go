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
	netpl "github.com/confluentinc/ccloud-sdk-go-v2/networking-privatelink/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

func privateLinkAttachmentResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   privateLinkAttachmentRead,
		CreateContext: privateLinkAttachmentCreate,
		DeleteContext: privateLinkAttachmentDelete,
		UpdateContext: privateLinkAttachmentUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: privateLinkAttachmentImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Description:  "The name of the Private Link Attachment.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramCloud: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider that hosts the resources to access with the PrivateLink attachment.",
				ValidateFunc: validation.StringInSlice(acceptedCloudProviders, false),
				Required:     true,
				ForceNew:     true,
			},
			paramRegion: {
				Type:         schema.TypeString,
				Description:  "The cloud service provider region where the resources to be accessed using the PrivateLink attachment are located.",
				ValidateFunc: validation.StringIsNotEmpty,
				Required:     true,
				ForceNew:     true,
			},
			paramEnvironment: environmentSchema(),
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

func privateLinkAttachmentRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattId := d.Id()
	if plattId == "" {
		return diag.Errorf("error reading Private Link Attachment: Private Link Attachment id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readPrivateLinkAttachmentAndSetAttributes(ctx, d, meta, plattId, environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Private Link Attachment %q: %s", plattId, createDescriptiveError(err)))
	}

	return nil
}

func readPrivateLinkAttachmentAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, plattId string, environmentId string) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Attachment %q=%q", paramId, plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	c := meta.(*Client)
	platt, resp, err := executePlattRead(c.netPLApiContext(ctx), c, plattId, environmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Private Link Attachment %q: %s", plattId, createDescriptiveError(err)), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Private Link Attachment %q in TF state because Private Link Attachment could not be found on the server", plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	plattJson, err := json.Marshal(platt)
	if err != nil {
		return nil, fmt.Errorf("error reading Private Link Attachment %q: error marshaling %#v to json: %s", plattId, platt, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Attachment %q: %s", plattId, plattJson), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	if _, err := setPrivateLinkAttachmentAttributes(d, platt); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Private Link Attachment %q", plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	return []*schema.ResourceData{d}, nil
}

func executePlattRead(ctx context.Context, c *Client, plattId string, environmentId string) (netpl.NetworkingV1PrivateLinkAttachment, *http.Response, error) {
	request := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachment(c.netPLApiContext(ctx), plattId).Environment(environmentId)
	return c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentExecute(request)
}

func privateLinkAttachmentCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	spec := netpl.NewNetworkingV1PrivateLinkAttachmentSpec()

	displayName := d.Get(paramDisplayName).(string)
	cloud := d.Get(paramCloud).(string)
	region := d.Get(paramRegion).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetEnvironment(netpl.ObjectReference{Id: environmentId})
	spec.SetCloud(cloud)
	spec.SetRegion(region)

	platt := netpl.NewNetworkingV1PrivateLinkAttachment()
	platt.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.CreateNetworkingV1PrivateLinkAttachment(c.netPLApiContext(ctx))
	request = request.NetworkingV1PrivateLinkAttachment(*platt)

	createPrivateLinkAttachmentRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Private Link Attachment: %s", createPrivateLinkAttachmentRequestJson))

	createdPlatt, _, err := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.CreateNetworkingV1PrivateLinkAttachmentExecute(request)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment %s", createDescriptiveError(err))
	}

	plattId := createdPlatt.GetId()
	d.SetId(plattId)

	if err := waitForPrivateLinkAttachmentToProvision(c.netPLApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Private Link Attachment %q to provision: %s", plattId, createDescriptiveError(err))
	}

	createdPrivateLinkAttachmentJson, err := json.Marshal(createdPlatt)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment %q: error marshaling %#v to json: %s", plattId, createdPlatt, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Private Link Attachment %q: %s", plattId, createdPrivateLinkAttachmentJson), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})
	return privateLinkAttachmentRead(ctx, d, meta)
}

func privateLinkAttachmentDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("deleting Private Link Attachment %q=%q", paramId, plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.DeleteNetworkingV1PrivateLinkAttachment(c.netPLApiContext(ctx), plattId).Environment(environmentId)
	_, err := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.DeleteNetworkingV1PrivateLinkAttachmentExecute(request)
	if err != nil {
		return diag.Errorf("error deleting Private Link Attachment %q: %s", plattId, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Private Link Attachment %q", plattId), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})

	return nil
}

func privateLinkAttachmentUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Private Link Attachment %q: only %q attribute can be updated for Private Link Attachment", d.Id(), paramDisplayName)
	}

	spec := netpl.NewNetworkingV1PrivateLinkAttachmentSpecUpdate()

	plattId := d.Id()
	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetEnvironment(netpl.ObjectReference{Id: environmentId})

	platt := netpl.NewNetworkingV1PrivateLinkAttachmentUpdate()
	platt.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.UpdateNetworkingV1PrivateLinkAttachment(c.netPLApiContext(ctx), plattId)
	request = request.NetworkingV1PrivateLinkAttachmentUpdate(*platt)

	updatePrivateLinkAttachmentRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Private Link Attachment: %s", updatePrivateLinkAttachmentRequestJson))

	updatedPlatt, _, err := c.netPLClient.PrivateLinkAttachmentsNetworkingV1Api.UpdateNetworkingV1PrivateLinkAttachmentExecute(request)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment, %s", createDescriptiveError(err))
	}

	updatedPrivateLinkAttachmentJson, err := json.Marshal(updatedPlatt)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment %q: error marshaling %#v to json: %s", plattId, updatedPlatt, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Private Link Attachment %q: %s", plattId, updatedPrivateLinkAttachmentJson), map[string]interface{}{privateLinkAttachmentLoggingKey: plattId})
	return privateLinkAttachmentRead(ctx, d, meta)
}

func privateLinkAttachmentImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Private Link Attachment %q", d.Id()), map[string]interface{}{privateLinkAttachmentLoggingKey: d.Id()})

	envIDAndPlattId := d.Id()
	parts := strings.Split(envIDAndPlattId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Private Link Attachment: invalid format: expected '<env ID>/<Private Link Attachment ID>'")
	}

	environmentId := parts[0]
	plattId := parts[1]
	d.SetId(plattId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readPrivateLinkAttachmentAndSetAttributes(ctx, d, meta, plattId, environmentId); err != nil {
		return nil, fmt.Errorf("error importing Private Link Attachment %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Private Link Attachment %q", d.Id()), map[string]interface{}{privateLinkAttachmentLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setPrivateLinkAttachmentAttributes(d *schema.ResourceData, platt netpl.NetworkingV1PrivateLinkAttachment) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, platt.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, platt.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramCloud, platt.Spec.GetCloud()); err != nil {
		return nil, err
	}
	if err := d.Set(paramRegion, platt.Spec.GetRegion()); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, platt.Metadata.GetResourceName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAws, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramAzure, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramGcp, []interface{}{}); err != nil {
		return nil, err
	}
	if err := d.Set(paramDnsDomain, platt.Status.GetDnsDomain()); err != nil {
		return nil, err
	}
	if platt.Status.GetCloud().NetworkingV1AwsPrivateLinkAttachmentStatus != nil {
		if err := setStringAttributeInListBlockOfSizeOne(paramAws, paramVpcEndpointServiceName, platt.Status.GetCloud().NetworkingV1AwsPrivateLinkAttachmentStatus.VpcEndpointService.GetVpcEndpointServiceName(), d); err != nil {
			return nil, err
		}
	} else if platt.Status.GetCloud().NetworkingV1AzurePrivateLinkAttachmentStatus != nil {
		if err := setAzurePrivateLinkService(d, platt.Status.GetCloud().NetworkingV1AzurePrivateLinkAttachmentStatus.GetPrivateLinkService()); err != nil {
			return nil, err
		}
	} else if platt.Status.GetCloud().NetworkingV1GcpPrivateLinkAttachmentStatus != nil {
		if err := setGcpServiceAttachments(d, platt.Status.GetCloud().NetworkingV1GcpPrivateLinkAttachmentStatus.GetServiceAttachments()); err != nil {
			return nil, err
		}
	}
	d.SetId(platt.GetId())

	return d, nil
}

func setAzurePrivateLinkService(d *schema.ResourceData, privateLinkService netpl.NetworkingV1AzurePrivateLinkService) error {
	return d.Set(paramAzure, []interface{}{map[string]interface{}{
		paramPrivateLinkServiceAlias:      privateLinkService.GetPrivateLinkServiceAlias(),
		paramPrivateLinkServiceResourceId: privateLinkService.GetPrivateLinkServiceResourceId(),
	}})
}

func setGcpServiceAttachments(d *schema.ResourceData, serviceAttachments []netpl.NetworkingV1GcpPscServiceAttachment) error {
	result := make([]interface{}, len(serviceAttachments))
	for i, t := range serviceAttachments {
		result[i] = map[string]interface{}{
			paramZone: t.GetZone(),
			paramPrivateServiceConnectServiceAttachment: t.GetPrivateServiceConnectServiceAttachment(),
		}
	}
	return d.Set(paramGcp, result)
}
