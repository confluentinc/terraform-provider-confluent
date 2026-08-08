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

// resource_switchover_pair_failover models the imperative `:failover` operation on a
// SwitchoverPair as a dedicated "action" resource. The Switchover API exposes failover as a
// POST .../{id}:failover call (not a mutable field on the pair), and the pair's own
// `active_member` is x-immutable (ForceNew), so it cannot be used to drive a failover. Modeling
// this as a separate resource keeps the confluent_switchover_pair resource's immutability intact
// and lets the caller express the failover inputs (active_member, failover_type) explicitly.
//
// Semantics:
//   - Create triggers the failover. All inputs are ForceNew, so changing any input recreates the
//     resource, which re-triggers a failover.
//   - Delete is a no-op (a failover cannot be undone by removing it from state; use RESTORE or a
//     new failover to change the active member back).
//   - Read refreshes only the computed `phase`; it never overwrites the user-provided inputs, to
//     avoid perpetual diffs if the pair's active member is changed out-of-band.

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	switchoverv1 "github.com/confluentinc/ccloud-sdk-go-v2-internal/switchover/v1"
)

func switchoverPairFailoverResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: switchoverPairFailoverCreate,
		ReadContext:   switchoverPairFailoverRead,
		DeleteContext: switchoverPairFailoverDelete,
		Schema: map[string]*schema.Schema{
			paramSwitchoverPairId: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The ID of the switchover pair to trigger a failover on.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramActiveMember: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "The name of the member to promote to active. Required when `failover_type` is `CLEAN` or `UNCLEAN`.",
			},
			paramFailoverType: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "CLEAN",
				Description:  "The failover semantics to apply: `CLEAN` (graceful, after replication lag reaches zero), `UNCLEAN` (immediate), or `RESTORE` (re-establish the cluster link after an unclean failover).",
				ValidateFunc: validation.StringInSlice([]string{"CLEAN", "UNCLEAN", "RESTORE"}, false),
			},
			paramPhase: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The lifecycle phase of the switchover pair after the failover was triggered (transitions to `SWITCHING`).",
			},
			paramEnvironment: environmentSchema(),
		},
	}
}

func switchoverPairFailoverCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	switchoverPairId := d.Get(paramSwitchoverPairId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	activeMember := d.Get(paramActiveMember).(string)
	failoverType := d.Get(paramFailoverType).(string)

	if (failoverType == "CLEAN" || failoverType == "UNCLEAN") && activeMember == "" {
		return diag.Errorf("error triggering switchover pair failover: %q is required when %q is %q or %q", paramActiveMember, paramFailoverType, "CLEAN", "UNCLEAN")
	}

	failoverSpec := switchoverv1.SwitchoverV1SwitchoverPairFailoverRequestSpec{
		Environment:  environmentId,
		FailoverType: switchoverv1.PtrString(failoverType),
	}
	if activeMember != "" {
		failoverSpec.ActiveMember = switchoverv1.PtrString(activeMember)
	}
	failoverRequest := switchoverv1.SwitchoverV1SwitchoverPairFailoverRequest{Spec: failoverSpec}

	failoverRequestJson, err := json.Marshal(failoverRequest)
	if err != nil {
		return diag.Errorf("error triggering switchover pair failover: error marshaling %#v to json: %s", failoverRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Triggering failover on switchover pair %q: %s", switchoverPairId, failoverRequestJson), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.FailoverSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), switchoverPairId).SwitchoverV1SwitchoverPairFailoverRequest(failoverRequest)
	pair, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error triggering switchover pair failover for %q: %s", switchoverPairId, createDescriptiveError(err, resp))
	}

	d.SetId(switchoverPairId)
	status := pair.GetStatus()
	if err := d.Set(paramPhase, status.GetPhase()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished triggering failover on switchover pair %q", switchoverPairId), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})
	return nil
}

func switchoverPairFailoverRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)
	switchoverPairId := d.Get(paramSwitchoverPairId).(string)
	environmentId := extractStringValueFromBlock(d, paramEnvironment, paramId)
	tflog.Debug(ctx, fmt.Sprintf("Reading switchover pair failover %q", d.Id()), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})

	req := c.switchoverV1Client.SwitchoverPairsSwitchoverV1Api.GetSwitchoverV1SwitchoverPair(c.switchoverV1ApiContext(ctx), switchoverPairId).Environment(environmentId)
	pair, resp, err := req.Execute()
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading switchover pair %q for failover: %s", switchoverPairId, createDescriptiveError(err, resp)), map[string]interface{}{switchoverPairLoggingKey: switchoverPairId})
		if isNonKafkaRestApiResourceNotFound(resp) && !d.IsNewResource() {
			d.SetId("")
			return nil
		}
		return diag.FromErr(fmt.Errorf("error reading switchover pair failover %q: %s", d.Id(), createDescriptiveError(err, resp)))
	}

	// Only refresh the computed phase; never overwrite the user-provided failover inputs.
	status := pair.GetStatus()
	if err := d.Set(paramPhase, status.GetPhase()); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func switchoverPairFailoverDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// A failover cannot be undone by removing it from state; deletion only drops the resource from
	// Terraform state. Use a RESTORE failover or a new failover to change the active member back.
	tflog.Debug(ctx, fmt.Sprintf("Removing switchover pair failover %q from state (no-op on the server)", d.Id()), map[string]interface{}{switchoverPairLoggingKey: d.Get(paramSwitchoverPairId).(string)})
	return nil
}
