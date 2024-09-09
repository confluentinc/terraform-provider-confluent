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

const (
	paramVpcEndpointId                        = "vpc_endpoint_id"
	paramPrivateEndpointResourceId            = "private_endpoint_resource_id"
	paramPrivateServiceConnectConnectionId    = "private_service_connect_connection_id"
	paramPrivateLinkAttachment                = "private_link_attachment"
	paramAwsPrivateLinkAttachmentConnection   = "AwsPrivateLinkAttachmentConnection"
	paramAzurePrivateLinkAttachmentConnection = "AzurePrivateLinkAttachmentConnection"
	paramGcpPrivateLinkAttachmentConnection   = "GcpPrivateLinkAttachmentConnection"
)

var acceptedPrivateLinkAttachmentConnectionKinds = []string{paramAws, paramAzure, paramGcp}

func privateLinkAttachmentConnectionResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   privateLinkAttachmentConnectionRead,
		CreateContext: privateLinkAttachmentConnectionCreate,
		DeleteContext: privateLinkAttachmentConnectionDelete,
		UpdateContext: privateLinkAttachmentConnectionUpdate,
		Importer: &schema.ResourceImporter{
			StateContext: privateLinkAttachmentConnectionImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Description:  "The name of the Private Link Attachment Connection.",
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramEnvironment:           environmentSchema(),
			paramPrivateLinkAttachment: privateLinkAttachmentSchema(),
			paramAws:                   awsPlattcSchema(),
			paramAzure:                 azurePlattcSchema(),
			paramGcp:                   gcpPlattcSchema(),
		},
	}
}

func awsPlattcSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramVpcEndpointId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Id of a VPC Endpoint that is connected to the VPC Endpoint service.",
				},
			},
		},
		ExactlyOneOf: acceptedPrivateLinkAttachmentConnectionKinds,
	}
}

func azurePlattcSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateEndpointResourceId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Resource Id of the PrivateEndpoint that is connected to the PrivateLink service.",
				},
			},
		},
		ExactlyOneOf: acceptedPrivateLinkAttachmentConnectionKinds,
	}
}

func gcpPlattcSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		ForceNew: true,
		MinItems: 1,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramPrivateServiceConnectConnectionId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "Id of the Private Service connection.",
				},
			},
		},
		ExactlyOneOf: acceptedPrivateLinkAttachmentConnectionKinds,
	}
}

func privateLinkAttachmentSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the private link attachment.",
				},
			},
		},
		Required:    true,
		MinItems:    1,
		MaxItems:    1,
		ForceNew:    true,
		Description: "The private_link_attachment to which this belongs.",
	}
}

func privateLinkAttachmentConnectionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattcId := d.Id()
	if plattcId == "" {
		return diag.Errorf("error reading Private Link Attachment Connection: Private Link Attachment Connection id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	if _, err := readPrivateLinkAttachmentConnectionAndSetAttributes(ctx, d, meta, plattcId, environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Private Link Attachment Connection %q: %s", plattcId, createDescriptiveError(err)))
	}

	return nil
}

func readPrivateLinkAttachmentConnectionAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, plattcId string, environmentId string) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Private Link Attachment Connection %q=%q", paramId, plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	c := meta.(*Client)
	plattc, resp, err := executePlattcRead(c.netPLApiContext(ctx), c, plattcId, environmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Private Link Attachment Connection %q: %s", plattcId, createDescriptiveError(err)), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Private Link Attachment Connection %q in TF state because Private Link Attachment Connection could not be found on the server", plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	plattcJson, err := json.Marshal(plattc)
	if err != nil {
		return nil, fmt.Errorf("error reading Private Link Attachment Connection %q: error marshaling %#v to json: %s", plattcId, plattc, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Private Link Attachment Connection %q: %s", plattcId, plattcJson), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	if _, err := setPrivateLinkAttachmentConnectionAttributes(d, plattc); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Private Link Attachment Connection %q", plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	return []*schema.ResourceData{d}, nil
}

func executePlattcRead(ctx context.Context, c *Client, plattcId string, environmentId string) (netpl.NetworkingV1PrivateLinkAttachmentConnection, *http.Response, error) {
	request := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentConnection(c.netPLApiContext(ctx), plattcId).Environment(environmentId)
	return c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.GetNetworkingV1PrivateLinkAttachmentConnectionExecute(request)
}

func privateLinkAttachmentConnectionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	spec := netpl.NewNetworkingV1PrivateLinkAttachmentConnectionSpec()

	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	plattId := extractStringValueFromBlock(d, paramPrivateLinkAttachment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetEnvironment(netpl.ObjectReference{Id: environmentId})
	spec.SetPrivateLinkAttachment(netpl.ObjectReference{Id: plattId})

	vpcEndpointId := extractStringValueFromBlock(d, paramAws, paramVpcEndpointId)
	privateEndpointResourceId := extractStringValueFromBlock(d, paramAzure, paramPrivateEndpointResourceId)
	privateServiceConnectConnectionId := extractStringValueFromBlock(d, paramGcp, paramPrivateServiceConnectConnectionId)

	if vpcEndpointId != "" {
		spec.SetCloud(netpl.NetworkingV1PrivateLinkAttachmentConnectionSpecCloudOneOf{NetworkingV1AwsPrivateLinkAttachmentConnection: netpl.NewNetworkingV1AwsPrivateLinkAttachmentConnection(paramAwsPrivateLinkAttachmentConnection, vpcEndpointId)})
	} else if privateEndpointResourceId != "" {
		spec.SetCloud(netpl.NetworkingV1PrivateLinkAttachmentConnectionSpecCloudOneOf{NetworkingV1AzurePrivateLinkAttachmentConnection: netpl.NewNetworkingV1AzurePrivateLinkAttachmentConnection(paramAzurePrivateLinkAttachmentConnection, privateEndpointResourceId)})
	} else if privateServiceConnectConnectionId != "" {
		spec.SetCloud(netpl.NetworkingV1PrivateLinkAttachmentConnectionSpecCloudOneOf{NetworkingV1GcpPrivateLinkAttachmentConnection: netpl.NewNetworkingV1GcpPrivateLinkAttachmentConnection(paramGcpPrivateLinkAttachmentConnection, privateServiceConnectConnectionId)})
	}

	plattc := netpl.NewNetworkingV1PrivateLinkAttachmentConnection()
	plattc.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.CreateNetworkingV1PrivateLinkAttachmentConnection(c.netPLApiContext(ctx))
	request = request.NetworkingV1PrivateLinkAttachmentConnection(*plattc)

	createPrivateLinkAttachmentConnectionRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment Connection: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Private Link Attachment Connection: %s", createPrivateLinkAttachmentConnectionRequestJson))

	createdPlattc, _, err := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.CreateNetworkingV1PrivateLinkAttachmentConnectionExecute(request)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment Connection %s", createDescriptiveError(err))
	}

	plattcId := createdPlattc.GetId()
	d.SetId(plattcId)

	if err := waitForPrivateLinkAttachmentConnectionToProvision(c.netPLApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Private Link Attachment Connection %q to provision: %s", plattcId, createDescriptiveError(err))
	}

	createdPrivateLinkAttachmentConnectionJson, err := json.Marshal(createdPlattc)
	if err != nil {
		return diag.Errorf("error creating Private Link Attachment Connection %q: error marshaling %#v to json: %s", plattcId, createdPlattc, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Private Link Attachment Connection %q: %s", plattcId, createdPrivateLinkAttachmentConnectionJson), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})
	return privateLinkAttachmentConnectionRead(ctx, d, meta)
}

func privateLinkAttachmentConnectionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	plattcId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("deleting Private Link Attachment Connection %q=%q", paramId, plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.DeleteNetworkingV1PrivateLinkAttachmentConnection(c.netPLApiContext(ctx), plattcId).Environment(environmentId)
	_, err := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.DeleteNetworkingV1PrivateLinkAttachmentConnectionExecute(request)
	if err != nil {
		return diag.Errorf("error deleting Private Link Attachment Connection %q: %s", plattcId, createDescriptiveError(err))
	}

	if err := waitForPrivateLinkAttachmentConnectionToBeDeleted(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Private Link Attachment Connection %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Private Link Attachment Connection %q", plattcId), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})

	return nil
}

func privateLinkAttachmentConnectionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangeExcept(paramDisplayName) {
		return diag.Errorf("error updating Private Link Attachment Connection %q: only %q attribute can be updated for Private Link Attachment Connection", d.Id(), paramDisplayName)
	}

	spec := netpl.NewNetworkingV1PrivateLinkAttachmentConnectionSpecUpdate()

	plattcId := d.Id()
	displayName := d.Get(paramDisplayName).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetEnvironment(netpl.ObjectReference{Id: environmentId})

	plattc := netpl.NewNetworkingV1PrivateLinkAttachmentConnectionUpdate()
	plattc.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.UpdateNetworkingV1PrivateLinkAttachmentConnection(c.netPLApiContext(ctx), plattcId)
	request = request.NetworkingV1PrivateLinkAttachmentConnectionUpdate(*plattc)

	updatePrivateLinkAttachmentConnectionRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment Connection: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Private Link Attachment Connection: %s", updatePrivateLinkAttachmentConnectionRequestJson))

	updatedPlattc, _, err := c.netPLClient.PrivateLinkAttachmentConnectionsNetworkingV1Api.UpdateNetworkingV1PrivateLinkAttachmentConnectionExecute(request)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment Connection, %s", createDescriptiveError(err))
	}

	updatedPrivateLinkAttachmentConnectionJson, err := json.Marshal(updatedPlattc)
	if err != nil {
		return diag.Errorf("error updating Private Link Attachment Connection %q: error marshaling %#v to json: %s", plattcId, updatedPlattc, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Private Link Attachment Connection %q: %s", plattcId, updatedPrivateLinkAttachmentConnectionJson), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: plattcId})
	return privateLinkAttachmentConnectionRead(ctx, d, meta)
}

func privateLinkAttachmentConnectionImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Private Link Attachment Connection %q", d.Id()), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: d.Id()})

	envIDAndPlattcId := d.Id()
	parts := strings.Split(envIDAndPlattcId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Private Link Attachment Connection: invalid format: expected '<env ID>/<Private Link Attachment Connection ID>'")
	}

	environmentId := parts[0]
	plattcId := parts[1]
	d.SetId(plattcId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readPrivateLinkAttachmentConnectionAndSetAttributes(ctx, d, meta, plattcId, environmentId); err != nil {
		return nil, fmt.Errorf("error importing Private Link Attachment Connection %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Private Link Attachment Connection %q", d.Id()), map[string]interface{}{privateLinkAttachmentConnectionLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setPrivateLinkAttachmentConnectionAttributes(d *schema.ResourceData, plattc netpl.NetworkingV1PrivateLinkAttachmentConnection) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, plattc.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, plattc.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramPrivateLinkAttachment, paramId, plattc.Spec.PrivateLinkAttachment.GetId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, plattc.Metadata.GetResourceName()); err != nil {
		return nil, err
	}
	if plattc.Spec.Cloud.NetworkingV1AwsPrivateLinkAttachmentConnection != nil {
		if err := setStringAttributeInListBlockOfSizeOne(paramAws, paramVpcEndpointId, plattc.Spec.Cloud.NetworkingV1AwsPrivateLinkAttachmentConnection.GetVpcEndpointId(), d); err != nil {
			return nil, err
		}
	} else if plattc.Spec.Cloud.NetworkingV1AzurePrivateLinkAttachmentConnection != nil {
		if err := setStringAttributeInListBlockOfSizeOne(paramAzure, paramPrivateEndpointResourceId, plattc.Spec.Cloud.NetworkingV1AzurePrivateLinkAttachmentConnection.GetPrivateEndpointResourceId(), d); err != nil {
			return nil, err
		}
	} else if plattc.Spec.Cloud.NetworkingV1GcpPrivateLinkAttachmentConnection != nil {
		if err := setStringAttributeInListBlockOfSizeOne(paramGcp, paramPrivateServiceConnectConnectionId, plattc.Spec.Cloud.NetworkingV1GcpPrivateLinkAttachmentConnection.GetPrivateServiceConnectConnectionId(), d); err != nil {
			return nil, err
		}
	}
	d.SetId(plattc.GetId())

	return d, nil
}
