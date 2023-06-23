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

func networkLinkServiceResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: networkLinkServiceCreate,
		ReadContext:   networkLinkServiceRead,
		UpdateContext: networkLinkServiceUpdate,
		DeleteContext: networkLinkServiceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: networkLinkServiceImport,
		},
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the Network Link Service, for example, `nls-a1b2c`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The display name of the Network Link Service.",
			},
			paramDescription: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramEnvironment: environmentSchema(),
			paramNetwork:     requiredNetworkSchema(),
			paramResourceName: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramAccept: acceptSchema(),
		},
	}
}

func acceptSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramEnvironments: {
					Type:     schema.TypeSet,
					Computed: true,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
				paramNetworks: {
					Type:     schema.TypeSet,
					Computed: true,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
		MaxItems: 1,
		Computed: true,
		Optional: true,
	}
}

func networkLinkServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {

	spec := net.NewNetworkingV1NetworkLinkServiceSpec()

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	networkId := extractStringValueFromBlock(d, paramNetwork, paramId)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetNetwork(net.EnvScopedObjectReference{Id: networkId})
	spec.SetEnvironment(net.GlobalObjectReference{Id: environmentId})

	accept := net.NewNetworkingV1NetworkLinkServiceAcceptPolicy()
	acceptNetworks := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAccept, paramNetworks)).(*schema.Set).List())
	accept.SetNetworks(acceptNetworks)

	acceptEnvironments := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAccept, paramEnvironments)).(*schema.Set).List())
	accept.SetEnvironments(acceptEnvironments)

	spec.SetAccept(*accept)

	nls := net.NewNetworkingV1NetworkLinkService()
	nls.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netClient.NetworkLinkServicesNetworkingV1Api.CreateNetworkingV1NetworkLinkService(c.netApiContext(ctx))
	request = request.NetworkingV1NetworkLinkService(*nls)

	createNetworkLinkServiceRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error creating Network Link Service: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Network Link Service: %s", createNetworkLinkServiceRequestJson))

	createdNLS, _, err := c.netClient.NetworkLinkServicesNetworkingV1Api.CreateNetworkingV1NetworkLinkServiceExecute(request)
	if err != nil {
		return diag.Errorf("error creating Network Link Service %s", createDescriptiveError(err))
	}

	nlsId := createdNLS.GetId()
	d.SetId(nlsId)

	if err := waitForNetworkLinkServiceToProvision(c.netApiContext(ctx), c, environmentId, d.Id()); err != nil {
		return diag.Errorf("error waiting for Network Link Service %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	createdNetworkLinkServiceJson, err := json.Marshal(createdNLS)
	if err != nil {
		return diag.Errorf("error creating Network Link Service %q: error marshaling %#v to json: %s", nlsId, createdNLS, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Network Link Service %q: %s", nlsId, createdNetworkLinkServiceJson), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})
	return networkLinkServiceRead(ctx, d, meta)
}

func networkLinkServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nlsId := d.Id()
	if nlsId == "" {
		return diag.Errorf("error reading Network Link Service: Network Link Service id is missing")
	}

	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	if environmentId == "" {
		return diag.Errorf("error reading Network Link Service: environment Id is missing")
	}

	if _, err := readNetworkLinkServiceAndSetAttributes(ctx, d, meta, nlsId, environmentId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Network Link Service %q: %s", nlsId, createDescriptiveError(err)))
	}

	return nil
}

func readNetworkLinkServiceAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, nlsId string, environmentId string) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Reading Network Link Service %q=%q", paramId, nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	c := meta.(*Client)
	nls, resp, err := executeNLSRead(ctx, c, nlsId, environmentId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Network Link Service %q: %s", nlsId, createDescriptiveError(err)), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Network Link Service %q in TF state because Network Link Service could not be found on the server", nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}

	nlsJson, err := json.Marshal(nls)
	if err != nil {
		return nil, fmt.Errorf("error reading Network Link Service %q: error marshaling %#v to json: %s", nlsId, nls, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Network Link Service %q: %s", nlsId, nlsJson), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	if _, err := setNetworkLinkServiceAttributes(d, nls); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Network Link Service %q", nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	return []*schema.ResourceData{d}, nil
}

func executeNLSRead(ctx context.Context, c *Client, nlsId string, environmentId string) (net.NetworkingV1NetworkLinkService, *http.Response, error) {
	request := c.netClient.NetworkLinkServicesNetworkingV1Api.GetNetworkingV1NetworkLinkService(c.netApiContext(ctx), nlsId).Environment(environmentId)
	return c.netClient.NetworkLinkServicesNetworkingV1Api.GetNetworkingV1NetworkLinkServiceExecute(request)
}

func networkLinkServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	nlsId := d.Id()
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	tflog.Debug(ctx, fmt.Sprintf("deleting Network Link Service %q=%q", paramId, nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	c := meta.(*Client)
	request := c.netClient.NetworkLinkServicesNetworkingV1Api.DeleteNetworkingV1NetworkLinkService(c.netApiContext(ctx), nlsId).Environment(environmentId)
	_, err := c.netClient.NetworkLinkServicesNetworkingV1Api.DeleteNetworkingV1NetworkLinkServiceExecute(request)
	if err != nil {
		return diag.Errorf("error deleting Network Link Service %q: %s", nlsId, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Network Link Service %q", nlsId), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})

	return nil
}

func networkLinkServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramAccept) {
		return diag.Errorf("error updating Network Link Service %q: only %q, %q and %q attributes can be updated for Network Link Endpoint", d.Id(), paramDisplayName, paramDescription, paramAccept)
	}

	spec := net.NewNetworkingV1NetworkLinkServiceSpecUpdate()

	nlsId := d.Id()
	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	spec.SetDisplayName(displayName)
	spec.SetDescription(description)
	spec.SetEnvironment(net.GlobalObjectReference{Id: environmentId})

	accept := net.NewNetworkingV1NetworkLinkServiceAcceptPolicy()
	acceptNetworks := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAccept, paramNetworks)).(*schema.Set).List())
	accept.SetNetworks(acceptNetworks)

	acceptEnvironments := convertToStringSlice(d.Get(fmt.Sprintf("%s.0.%s", paramAccept, paramEnvironments)).(*schema.Set).List())
	accept.SetEnvironments(acceptEnvironments)

	spec.SetAccept(*accept)

	nls := net.NewNetworkingV1NetworkLinkServiceUpdate()
	nls.SetSpec(*spec)

	c := meta.(*Client)
	request := c.netClient.NetworkLinkServicesNetworkingV1Api.UpdateNetworkingV1NetworkLinkService(c.netApiContext(ctx), nlsId)
	request = request.NetworkingV1NetworkLinkServiceUpdate(*nls)

	updateNetworkLinkServiceRequestJson, err := json.Marshal(request)
	if err != nil {
		return diag.Errorf("error updating Network Link Service: error marshaling %#v to json: %s", request, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating new Network Link Service: %s", updateNetworkLinkServiceRequestJson))

	updatedNLS, _, err := c.netClient.NetworkLinkServicesNetworkingV1Api.UpdateNetworkingV1NetworkLinkServiceExecute(request)
	if err != nil {
		return diag.Errorf("error updating Network Link Service, %s", createDescriptiveError(err))
	}

	updatedNetworkLinkServiceJson, err := json.Marshal(updatedNLS)
	if err != nil {
		return diag.Errorf("error updating Network Link Service %q: error marshaling %#v to json: %s", nlsId, updatedNLS, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Network Link Service %q: %s", nlsId, updatedNetworkLinkServiceJson), map[string]interface{}{networkLinkServiceLoggingKey: nlsId})
	return networkLinkServiceRead(ctx, d, meta)
}

func networkLinkServiceImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Network Link Service %q", d.Id()), map[string]interface{}{networkLinkServiceLoggingKey: d.Id()})

	envIDAndNLSId := d.Id()
	parts := strings.Split(envIDAndNLSId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Network Link Service: invalid format: expected '<env ID>/<Network Link Service ID>'")
	}

	environmentId := parts[0]
	nlsId := parts[1]
	d.SetId(nlsId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readNetworkLinkServiceAndSetAttributes(ctx, d, meta, nlsId, environmentId); err != nil {
		return nil, fmt.Errorf("error importing Network Link Service %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Network Link Service %q", d.Id()), map[string]interface{}{networkLinkServiceLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setNetworkLinkServiceAttributes(d *schema.ResourceData, nls net.NetworkingV1NetworkLinkService) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, nls.Spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, nls.Spec.GetDescription()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, nls.Spec.Environment.GetId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramNetwork, paramId, nls.Spec.Network.GetId(), d); err != nil {
		return nil, err
	}
	if err := d.Set(paramResourceName, nls.Metadata.GetResourceName()); err != nil {
		return nil, err
	}
	if err := setAccept(d, nls); err != nil {
		return nil, err
	}
	d.SetId(nls.GetId())

	return d, nil
}

func setAccept(d *schema.ResourceData, nls net.NetworkingV1NetworkLinkService) error {
	return d.Set(paramAccept, []interface{}{map[string]interface{}{
		paramNetworks:     nls.Spec.Accept.GetNetworks(),
		paramEnvironments: nls.Spec.Accept.GetEnvironments(),
	}})
}
