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
	net "github.com/confluentinc/ccloud-sdk-go-v2/networking/v1"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
	"strings"
)

const (
	stateDeProvisioning = "DEPROVISIONING"
	stateInactive       = "INACTIVE"
)

func networkLinkEndpointResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: networkLinkEndpointCreate,
		ReadContext:   networkLinkEndpointRead,
		UpdateContext: networkLinkEndpointUpdate,
		DeleteContext: networkLinkEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: networkLinkEndpointImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the Network Link Endpoint, for example, `nle-a1b2c`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The display name of the Network Link Endpoint.",
			},
			paramDescription: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramEnvironment:        environmentSchema(),
			paramNetwork:            requiredNetworkSchema(),
			paramNetworkLinkService: networkLinkServiceSchema(),
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func networkLinkServiceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the Network Link Endpoint.",
				},
			},
		},
		Required: true,
		MinItems: 1,
		MaxItems: 1,
		ForceNew: true,
	}
}

func networkLinkEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	spec := net.NewNetworkingV1NetworkLinkEndpointSpec()

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	nlsId := extractStringValueFromBlock(d, paramNetworkLinkService, paramId)

	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetNetwork(net.EnvScopedObjectReference{Id: networkId})
	spec.SetEnvironment(net.GlobalObjectReference{Id: environmentId})
	spec.SetNetworkLinkService(net.EnvScopedObjectReference{Id: nlsId})

	nle := net.NewNetworkingV1NetworkLinkEndpoint()
	nle.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netClient.NetworkLinkEndpointsNetworkingV1Api.CreateNetworkingV1NetworkLinkEndpoint(c.netApiContext(ctx))
	request = request.NetworkingV1NetworkLinkEndpoint(*nle)

	createNetworkLinkEndpointRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Network Link Endpoint: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Network Link Endpoint: %s", createNetworkLinkEndpointRequestJson))

	createdNLE, _, err := c.netClient.NetworkLinkEndpointsNetworkingV1Api.CreateNetworkingV1NetworkLinkEndpointExecute(request)
	if err != nil {
		return diag.Errorf("error creating Network Link Endpoint %s", createDescriptiveError(err))
	}

	nleId := createdNLE.GetId()
	d.SetId(nleId)

	if err := waitForNetworkLinkEndpointToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Network Link Endpoint %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdNetworkLinkEndpointJson, err := json.Marshal(createdNLE)
	if err != nil {
		return diag.Errorf("error creating Network Link Endpoint %q: error marshaling %#v to json: %s", nleId, createdNLE, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Network Link Endpoint %q: %s", nleId, createdNetworkLinkEndpointJson), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})
	return networkLinkEndpointRead(ctx, d, meta)
}

func networkLinkEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nleId := d.Id()
	if nleId == "" {
		return diag.Errorf("error reading Network Link Endpoint: Network Link Endpoint id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading Network Link Endpoint: environment Id is missing")
	}

	if _, err := readNetworkLinkEndpointAndSetAttributes(ctx, d, meta, nleId, environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Network Link Endpoint %q: %s", nleId, createDescriptiveError(err)))
	}

	return nil
}

func executeNLERead(ctx context.Context, c *Client, nleId string, environmentId string) (net.NetworkingV1NetworkLinkEndpoint, *http.Response, error) {
	request := c.netClient.NetworkLinkEndpointsNetworkingV1Api.GetNetworkingV1NetworkLinkEndpoint(c.netApiContext(ctx), nleId).Environment(environmentId)
	return c.netClient.NetworkLinkEndpointsNetworkingV1Api.GetNetworkingV1NetworkLinkEndpointExecute(request)
}

func readNetworkLinkEndpointAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, nleId string, environmentId string) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Network Link Endpoint %q=%q", paramId, nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	c := meta.(*Client)
	nle, resp, err := executeNLERead(c.netApiContext(ctx), c, nleId, environmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Network Link Endpoint %q: %s", nleId, createDescriptiveError(err)), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Network Link Endpoint %q in TF state because Network Link Endpoint could not be found on the server", nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	nleJson, err := json.Marshal(nle)
	if err != nil {
		return nil, fmt.Errorf("error reading Network Link Endpoint %q: error marshaling %#v to json: %s", nleId, nle, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Network Link Endpoint %q: %s", nleId, nleJson), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	if _, err := setNetworkLinkEndpointAttributes(d, nle); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Network Link Endpoint %q", nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	return []*schema.ResourceData{d}, nil
}

func networkLinkEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nleId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("deleting Network Link Endpoint %q=%q", paramId, nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	c := meta.(*Client)
	request := c.netClient.NetworkLinkEndpointsNetworkingV1Api.DeleteNetworkingV1NetworkLinkEndpoint(c.netApiContext(ctx), nleId).Environment(environmentId)
	_, err := c.netClient.NetworkLinkEndpointsNetworkingV1Api.DeleteNetworkingV1NetworkLinkEndpointExecute(request)
	if err != nil {
		return diag.Errorf("error deleting Network Link Endpoint %q: %s", nleId, createDescriptiveError(err))
	}

	if err := waitForNetworkLinkEndpointToBeDeleted(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Network Link Endpoint %q to be deleted: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Network Link Endpoint %q", nleId), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})

	return nil
}

func networkLinkEndpointUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription) {
		return diag.Errorf("error updating Network Link Endpoint %q: only %q and %q attributes can be updated for Network Link Endpoint", d.Id(), paramDisplayName, paramDescription)
	}

	spec := net.NewNetworkingV1NetworkLinkEndpointSpecUpdate()

	nleId := d.Id()
	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetEnvironment(net.GlobalObjectReference{Id: environmentId})

	nle := net.NewNetworkingV1NetworkLinkEndpointUpdate()
	nle.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netClient.NetworkLinkEndpointsNetworkingV1Api.UpdateNetworkingV1NetworkLinkEndpoint(c.netApiContext(ctx), nleId)
	request = request.NetworkingV1NetworkLinkEndpointUpdate(*nle)

	updateNetworkLinkEndpointRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Network Link Endpoint: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Network Link Endpoint: %s", updateNetworkLinkEndpointRequestJson))

	updatedNLE, _, err := c.netClient.NetworkLinkEndpointsNetworkingV1Api.UpdateNetworkingV1NetworkLinkEndpointExecute(request)
	if err != nil {
		return diag.Errorf("error updating Network Link Endpoint, %s", createDescriptiveError(err))
	}

	updatedNetworkLinkEndpointJson, err := json.Marshal(updatedNLE)
	if err != nil {
		return diag.Errorf("error updating Network Link Endpoint %q: error marshaling %#v to json: %s", nleId, updatedNLE, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Network Link Endpoint %q: %s", nleId, updatedNetworkLinkEndpointJson), map[string]interface{}{networkLinkEndpointLoggingKey: nleId})
	return networkLinkEndpointRead(ctx, d, meta)
}

func networkLinkEndpointImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Network Link Endpoint %q", d.Id()), map[string]interface{}{networkLinkEndpointLoggingKey: d.Id()})

	envIDAndNLEId := d.Id()
	parts := strings.Split(envIDAndNLEId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Network Link Endpoint: invalid format: expected '<env ID>/<Network Link Endpoint ID>'")
	}

	environmentId := parts[0]
	nleId := parts[1]
	d.SetId(nleId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readNetworkLinkEndpointAndSetAttributes(ctx, d, meta, nleId, environmentId); err != nil {
		return nil, fmt.Errorf("error importing Network Link Endpoint %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Network Link Endpoint %q", d.Id()), map[string]interface{}{networkLinkEndpointLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setNetworkLinkEndpointAttributes(d *schema.ResourceData, nle net.NetworkingV1NetworkLinkEndpoint) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, nle.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, nle.Spec.GetDescription()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, nle.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, nle.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramNetworkLinkService, paramId, nle.Spec.NetworkLinkService.GetId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, nle.Metadata.GetResourceName()); err != nil {
		return nil, err
	}
	d.SetId(nle.GetId())

	return d, nil
}
