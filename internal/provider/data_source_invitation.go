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
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	paramAcceptedAt    = "accepted_at"
	paramExpiresAt     = "expires_at"
	paramAuthType      = "auth_type"
	paramUser          = "user"
	paramCreator       = "creator"
	paramAllowDeletion = "allow_deletion"
	statusAccepted     = "INVITE_STATUS_ACCEPTED"
)

func invitationDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: invitationDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the invitation, for example, `dlz-f3a90de`.",
			},
			paramEmail: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramExpiresAt: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramAcceptedAt: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramStatus: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramAuthType: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramUser:    userSchema(),
			paramCreator: userSchema(),
		},
	}
}

func invitationDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	invitationId := d.Get(paramId).(string)

	if invitationId != "" {
		return invitationDataSourceReadUsingId(ctx, d, meta, invitationId)
	} else {
		return diag.Errorf("error reading invitation: invitation id is missing")
	}
}

func invitationDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, invitationId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading invitation %q=%q", paramId, invitationId), map[string]interface{}{invitationLoggingKey: invitationId})

	c := meta.(*Client)
	invitation, resp, err := executeInvitationRead(c.iamApiContext(ctx), c, invitationId)
	if err != nil {
		return diag.Errorf("error reading invitation %q: %s", invitationId, createDescriptiveError(err, resp))
	}
	invitationJson, err := json.Marshal(invitation)
	if err != nil {
		return diag.Errorf("error reading invitation %q: error marshaling %#v to json: %s", invitationId, invitation, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched invitation %q: %s", invitationId, invitationJson), map[string]interface{}{invitationLoggingKey: invitationId})

	if _, err := setInvitationAttributes(d, invitation); err != nil {
		return diag.FromErr(createDescriptiveError(err, resp))
	}
	return nil
}
