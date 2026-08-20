// Copyright 2021 Confluent Inc. All Rights Reserved.
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
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	switchoverv1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/switchover/v1"
)

const switchoverEndpointLoggingKey = "switchover_endpoint_id"

func switchoverEndpointResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: switchoverEndpointCreate,
		ReadContext:   switchoverEndpointRead,
		UpdateContext: switchoverEndpointUpdate,
		DeleteContext: switchoverEndpointDelete,
		Importer: &schema.ResourceImporter{
			StateContext: switchoverEndpointImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human-readable name for the switchover endpoint.",
				ValidateFunc: validation.StringLenBetween(1, 256),
			},
			paramSwitchoverPairId: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The ID of the switchover pair this endpoint is bound to.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramTarget: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The name of the endpoint that should be active. For stateful pairs the control plane owns this value (it follows the pair's active member); on create it may be provided as an initial value. Must match one of the `endpoints[].name` values.",
			},
			paramEndpoints: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MinItems:    2,
				MaxItems:    2,
				Description: "The endpoint definitions, one per side (e.g. west/east). Must contain exactly 2 entries.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "A logical name for this endpoint side (e.g. \"west-platt\"), unique within the resource.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramHostname: {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resolved hostname for this endpoint. Set by the Switchover service.",
						},
						paramEndpointFilter: {
							Type:        schema.TypeList,
							Required:    true,
							ForceNew:    true,
							MinItems:    1,
							MaxItems:    1,
							Description: "Filter criteria that identify a network endpoint for this side of the pair.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									paramType: {
										Type:         schema.TypeString,
										Required:     true,
										ForceNew:     true,
										Description:  "Whether the endpoint is private or public.",
										ValidateFunc: validation.StringInSlice([]string{"private", "public"}, false),
									},
									paramNetworkId: {
										Type:        schema.TypeString,
										Optional:    true,
										ForceNew:    true,
										Description: "The network ID, when applicable.",
									},
									paramAccessPoint: {
										Type:        schema.TypeString,
										Optional:    true,
										ForceNew:    true,
										Description: "The network access point ID, for access-point (PNI) endpoints.",
									},
								},
							},
						},
					},
				},
			},
			paramPhase: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The lifecycle phase of the switchover endpoint.",
			},
			paramEnvironment: environmentSchema(),
		},
	}
}

func switchoverEndpointCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	switchoverPairId := d.Get(paramSwitchoverPairId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	endpoints := buildSwitchoverEndpoints(d)

	spec := &switchoverv1.SwitchoverV1SwitchoverEndpointSpec{
		DisplayName:      switchoverv1.PtrString(displayName),
		Environment:      switchoverv1.PtrString(environmentId),
		SwitchoverPairId: switchoverv1.PtrString(switchoverPairId),
		Endpoints:        &endpoints,
	}
	if target := d.Get(paramTarget).(string); target != "" {
		spec.Target = switchoverv1.PtrString(target)
	}

	createRequest := switchoverv1.SwitchoverV1SwitchoverEndpoint{Spec: spec}

	createRequestJson, err := json.Marshal(createRequest)
	if err != nil {
		return diag.Errorf("error creating switchover endpoint: error marshaling %#v to json: %s", createRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new switchover endpoint: %s", createRequestJson))

	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.CreateSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx)).SwitchoverV1SwitchoverEndpoint(createRequest)
	createdEndpoint, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating switchover endpoint: %s", createDescriptiveError(err, resp))
	}

	d.SetId(createdEndpoint.GetId())
	tflog.Debug(ctx, fmt.Sprintf("Finished creating switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})
	return switchoverEndpointRead(ctx, d, meta)
}

func switchoverEndpointRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})

	if _, err := readSwitchoverEndpointAndSetAttributes(ctx, d, meta, environmentId, d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("error reading switchover endpoint %q: %s", d.Id(), createDescriptiveError(err)))
	}
	return nil
}

func switchoverEndpointUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName) {
		return diag.Errorf("error updating switchover endpoint %q: only %q can be updated", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	updateRequest := switchoverv1.SwitchoverV1SwitchoverEndpointUpdateRequest{
		Spec: switchoverv1.SwitchoverV1SwitchoverEndpointUpdateRequestSpec{
			DisplayName: switchoverv1.PtrString(d.Get(paramDisplayName).(string)),
		},
	}

	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.UpdateSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx), d.Id()).Environment(environmentId).SwitchoverV1SwitchoverEndpointUpdateRequest(updateRequest)
	if _, resp, err := req.Execute(); err != nil {
		return diag.Errorf("error updating switchover endpoint %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished updating switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})
	return switchoverEndpointRead(ctx, d, meta)
}

func switchoverEndpointDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Deleting switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})

	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.DeleteSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx), d.Id()).Environment(environmentId)
	if resp, err := req.Execute(); err != nil {
		return diag.Errorf("error deleting switchover endpoint %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})
	return nil
}

func switchoverEndpointImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})

	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing switchover endpoint: invalid format: expected '<env ID>/<switchover endpoint ID>'")
	}
	environmentId := parts[0]
	switchoverEndpointId := parts[1]
	d.SetId(switchoverEndpointId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSwitchoverEndpointAndSetAttributes(ctx, d, meta, environmentId, switchoverEndpointId); err != nil {
		return nil, fmt.Errorf("error importing switchover endpoint %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing switchover endpoint %q", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSwitchoverEndpointAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, id string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	req := c.switchoverV1Client.SwitchoverEndpointsSwitchoverV1Api.GetSwitchoverV1SwitchoverEndpoint(c.switchoverV1ApiContext(ctx), id).Environment(environmentId)
	endpoint, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading switchover endpoint %q: %s", id, createDescriptiveError(err, resp)), map[string]interface{}{switchoverEndpointLoggingKey: id})
		if isNonKafkaRestApiResourceNotFound(resp) && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing switchover endpoint %q in TF state because it could not be found on the server", d.Id()), map[string]interface{}{switchoverEndpointLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}
		return nil, err
	}

	if _, err := setSwitchoverEndpointAttributes(d, endpoint, environmentId); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading switchover endpoint %q", id), map[string]interface{}{switchoverEndpointLoggingKey: id})
	return []*schema.ResourceData{d}, nil
}

func setSwitchoverEndpointAttributes(d *schema.ResourceData, endpoint switchoverv1.SwitchoverV1SwitchoverEndpoint, environmentId string) (*schema.ResourceData, error) {
	spec := endpoint.GetSpec()
	if err := d.Set(paramDisplayName, spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSwitchoverPairId, spec.GetSwitchoverPairId()); err != nil {
		return nil, err
	}
	if err := d.Set(paramTarget, spec.GetTarget()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEndpoints, flattenSwitchoverEndpoints(spec.GetEndpoints())); err != nil {
		return nil, err
	}
	status := endpoint.GetStatus()
	if err := d.Set(paramPhase, status.GetPhase()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, environmentId, d); err != nil {
		return nil, err
	}

	d.SetId(endpoint.GetId())
	return d, nil
}

func buildSwitchoverEndpoints(d *schema.ResourceData) []switchoverv1.SwitchoverV1EndpointConfig {
	rawEndpoints := d.Get(paramEndpoints).([]interface{})
	endpoints := make([]switchoverv1.SwitchoverV1EndpointConfig, len(rawEndpoints))
	for i, raw := range rawEndpoints {
		block := raw.(map[string]interface{})
		filter := switchoverv1.SwitchoverV1EndpointFilter{}
		if rawFilters, ok := block[paramEndpointFilter].([]interface{}); ok && len(rawFilters) == 1 {
			filterBlock := rawFilters[0].(map[string]interface{})
			filter.Type = filterBlock[paramType].(string)
			if networkId, ok := filterBlock[paramNetworkId].(string); ok && networkId != "" {
				filter.NetworkId = switchoverv1.PtrString(networkId)
			}
			if accessPoint, ok := filterBlock[paramAccessPoint].(string); ok && accessPoint != "" {
				filter.AccessPoint = switchoverv1.PtrString(accessPoint)
			}
		}
		endpoints[i] = switchoverv1.SwitchoverV1EndpointConfig{
			Name:           block[paramName].(string),
			EndpointFilter: filter,
		}
	}
	return endpoints
}

func flattenSwitchoverEndpoints(endpoints []switchoverv1.SwitchoverV1EndpointConfig) []interface{} {
	result := make([]interface{}, len(endpoints))
	for i, endpoint := range endpoints {
		filter := endpoint.GetEndpointFilter()
		result[i] = map[string]interface{}{
			paramName:     endpoint.GetName(),
			paramHostname: endpoint.GetHostname(),
			paramEndpointFilter: []interface{}{map[string]interface{}{
				paramType:        filter.GetType(),
				paramNetworkId:   filter.GetNetworkId(),
				paramAccessPoint: filter.GetAccessPoint(),
			}},
		}
	}
	return result
}
