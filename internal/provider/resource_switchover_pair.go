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

const switchoverPairLoggingKey = "switchover_pair_id"

func switchoverPairResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: switchoverPairCreate,
		ReadContext:   switchoverPairRead,
		UpdateContext: switchoverPairUpdate,
		DeleteContext: switchoverPairDelete,
		Importer: &schema.ResourceImporter{
			StateContext: switchoverPairImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human-readable name for the switchover pair.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramMembers: {
				Type:        schema.TypeList,
				Required:    true,
				ForceNew:    true,
				MinItems:    2,
				MaxItems:    2,
				Description: "The two clusters participating in this switchover pair. Must contain exactly 2 members.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						paramName: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "A logical name for this member (e.g. \"west\" or \"east\"), unique within the pair.",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramMemberId: {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							Description:  "The ID of the cluster this member represents (e.g. an `lkc-` Kafka cluster ID).",
							ValidateFunc: validation.StringIsNotEmpty,
						},
						paramEnvId: {
							Type:        schema.TypeString,
							Optional:    true,
							ForceNew:    true,
							Description: "The environment ID of the member's cluster. Defaults to the switchover pair's environment when omitted.",
						},
					},
				},
			},
			paramActiveMember: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The name of the member that starts as active; must match one of the `members[].name` values. Use a failover operation to change the active member after creation.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFailoverType: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The failover semantics most recently applied to this pair. Defaults to `CLEAN` until a failover has been triggered.",
			},
			paramPhase: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The lifecycle phase of the switchover pair.",
			},
			paramEnvironment: environmentSchema(),
		},
	}
}

func switchoverPairCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	activeMember := d.Get(paramActiveMember).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	members := buildSwitchoverPairMembers(d)

	createRequest := switchoverv1.SwitchoverV1SwitchoverPair{
		Spec: &switchoverv1.SwitchoverV1SwitchoverPairSpec{
			DisplayName:  switchoverv1.PtrString(displayName),
			Environment:  switchoverv1.PtrString(environmentId),
			Members:      &members,
			ActiveMember: switchoverv1.PtrString(activeMember),
		},
	}

	createRequestJson, err := json.Marshal(createRequest)
	if err != nil {
		return diag.Errorf("error creating switchover pair: error marshaling %#v to json: %s", createRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new switchover pair: %s", createRequestJson))

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.CreateSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx)).SwitchoverV1SwitchoverPair(createRequest)
	createdPair, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating switchover pair: %s", createDescriptiveError(err, resp))
	}

	d.SetId(createdPair.GetId())
	tflog.Debug(ctx, fmt.Sprintf("Finished creating switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})
	return switchoverPairRead(ctx, d, meta)
}

func switchoverPairRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})

	if _, err := readSwitchoverPairAndSetAttributes(ctx, d, meta, environmentId, d.Id()); err != nil {
		return diag.FromErr(fmt.Errorf("error reading switchover pair %q: %s", d.Id(), createDescriptiveError(err)))
	}
	return nil
}

func switchoverPairUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName) {
		return diag.Errorf("error updating switchover pair %q: only %q can be updated", d.Id(), paramDisplayName)
	}

	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)

	updateRequest := switchoverv1.SwitchoverV1SwitchoverPairUpdateRequest{
		Spec: switchoverv1.SwitchoverV1SwitchoverPairUpdateRequestSpec{
			DisplayName: switchoverv1.PtrString(d.Get(paramDisplayName).(string)),
		},
	}

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.UpdateSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), d.Id()).Environment(environmentId).SwitchoverV1SwitchoverPairUpdateRequest(updateRequest)
	if _, resp, err := req.Execute(); err != nil {
		return diag.Errorf("error updating switchover pair %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished updating switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})
	return switchoverPairRead(ctx, d, meta)
}

func switchoverPairDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Deleting switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.DeleteSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), d.Id()).Environment(environmentId)
	if resp, err := req.Execute(); err != nil {
		return diag.Errorf("error deleting switchover pair %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})
	return nil
}

func switchoverPairImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})

	parts := strings.Split(d.Id(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing switchover pair: invalid format: expected '<env ID>/<switchover pair ID>'")
	}
	environmentId := parts[0]
	switchoverPairId := parts[1]
	d.SetId(switchoverPairId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readSwitchoverPairAndSetAttributes(ctx, d, meta, environmentId, switchoverPairId); err != nil {
		return nil, fmt.Errorf("error importing switchover pair %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing switchover pair %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func readSwitchoverPairAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, environmentId, id string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.GetSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), id).Environment(environmentId)
	pair, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading switchover pair %q: %s", id, createDescriptiveError(err, resp)), map[string]interface{}{switchoverPairLoggingKey: id})
		if isNonKafkaRestApiResourceNotFound(resp) && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing switchover pair %q in TF state because it could not be found on the server", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}
		return nil, err
	}

	if _, err := setSwitchoverPairAttributes(d, pair, environmentId); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading switchover pair %q", id), map[string]interface{}{switchoverPairLoggingKey: id})
	return []*schema.ResourceData{d}, nil
}

func setSwitchoverPairAttributes(d *schema.ResourceData, pair switchoverv1.SwitchoverV1SwitchoverPair, environmentId string) (*schema.ResourceData, error) {
	spec := pair.GetSpec()
	if err := d.Set(paramDisplayName, spec.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramActiveMember, spec.GetActiveMember()); err != nil {
		return nil, err
	}
	if err := d.Set(paramFailoverType, spec.GetFailoverType()); err != nil {
		return nil, err
	}
	if err := d.Set(paramMembers, flattenSwitchoverPairMembers(spec.GetMembers())); err != nil {
		return nil, err
	}
	status := pair.GetStatus()
	if err := d.Set(paramPhase, status.GetPhase()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramEnvironment, paramId, environmentId, d); err != nil {
		return nil, err
	}

	d.SetId(pair.GetId())
	return d, nil
}

func buildSwitchoverPairMembers(d *schema.ResourceData) []switchoverv1.SwitchoverV1SwitchoverPairMember {
	rawMembers := d.Get(paramMembers).([]interface{})
	members := make([]switchoverv1.SwitchoverV1SwitchoverPairMember, len(rawMembers))
	for i, raw := range rawMembers {
		block := raw.(map[string]interface{})
		member := switchoverv1.SwitchoverV1SwitchoverPairMember{
			Name:     block[paramName].(string),
			MemberId: block[paramMemberId].(string),
		}
		if envId, ok := block[paramEnvId].(string); ok && envId != "" {
			member.EnvId = switchoverv1.PtrString(envId)
		}
		members[i] = member
	}
	return members
}

func flattenSwitchoverPairMembers(members []switchoverv1.SwitchoverV1SwitchoverPairMember) []interface{} {
	result := make([]interface{}, len(members))
	for i, member := range members {
		result[i] = map[string]interface{}{
			paramName:     member.GetName(),
			paramMemberId: member.GetMemberId(),
			paramEnvId:    member.GetEnvId(),
		}
	}
	return result
}
