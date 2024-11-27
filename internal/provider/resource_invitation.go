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
	iam "github.com/confluentinc/ccloud-sdk-go-v2-internal/iam/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

func invitationResource() *schema.Resource {
	return &schema.Resource{
		ReadContext:   invitationRead,
		CreateContext: invitationCreate,
		UpdateContext: invitationUpdate,
		DeleteContext: invitationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: invitationImport,
		},
		Schema: map[string]*schema.Schema{
			paramEmail: {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
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
				Optional: true,
			},
			paramUser:    userSchema(),
			paramCreator: userSchema(),
			paramAllowDeletion: {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func userSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}

func invitationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	invitationId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Reading invitation %q=%q", paramId, invitationId), map[string]interface{}{invitationloggingKey: invitationId})
	if _, err := readInvitationAndSetAttributes(ctx, d, meta, invitationId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading invitation %q: %s", invitationId, createDescriptiveError(err)))
	}

	return nil
}

func invitationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	invitationId := d.Id()

	tflog.Debug(ctx, fmt.Sprintf("Imporing invitation %q=%q", paramId, invitationId), map[string]interface{}{invitationloggingKey: invitationId})
	d.MarkNewResource()
	if _, err := readInvitationAndSetAttributes(ctx, d, meta, invitationId); err != nil {
		return nil, fmt.Errorf("error importing invitation %q: %s", invitationId, createDescriptiveError(err))
	}

	return []*schema.ResourceData{d}, nil
}

func readInvitationAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, invitationId string) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Reading invitation %q=%q", paramId, invitationId), map[string]interface{}{invitationloggingKey: invitationId})

	c := meta.(*Client)
	invitation, resp, err := executeInvitationRead(c.iamApiContext(ctx), c, invitationId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading invitation %q: %s", invitationId, createDescriptiveError(err)), map[string]interface{}{invitationloggingKey: invitationId})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing invitation %q in TF state because invitation could not be found on the server", invitationId), map[string]interface{}{invitationloggingKey: invitationId})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	invitationJson, err := json.Marshal(invitation)
	if err != nil {
		return nil, fmt.Errorf("error reading invitation %q: error marshaling %#v to json: %s", invitationId, invitation, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched invitation %q: %s", invitationId, invitationJson), map[string]interface{}{invitationloggingKey: invitationId})

	if _, err := setInvitationAttributes(d, invitation); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading invitation %q", invitationId), map[string]interface{}{invitationloggingKey: invitationId})

	return []*schema.ResourceData{d}, nil
}

func invitationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	email := d.Get(paramEmail).(string)
	authType := d.Get(paramAuthType).(string)

	createInvitationRequest := iam.IamV2Invitation{}
	createInvitationRequest.SetEmail(email)

	if len(authType) > 0 {
		createInvitationRequest.SetAuthType(authType)
	}

	createInvitationRequestJson, err := json.Marshal(createInvitationRequest)
	if err != nil {
		return diag.Errorf("error creating invitation: error marshaling %#v to json: %s", createInvitationRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new invitation: %s", createInvitationRequestJson))

	createdInvitation, _, err := executeInvitationCreate(c.iamApiContext(ctx), c, &createInvitationRequest)
	if err != nil {
		return diag.Errorf("error creating invitation %s", createDescriptiveError(err))
	}
	d.SetId(createdInvitation.GetId())

	createdInvitationJson, err := json.Marshal(createdInvitation)
	if err != nil {
		return diag.Errorf("error creating invitation %q: error marshaling %#v to json: %s", d.Id(), createdInvitation, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating invitation %q: %s", d.Id(), createdInvitationJson), map[string]interface{}{invitationloggingKey: d.Id()})

	return invitationRead(ctx, d, meta)
}
func invitationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramAllowDeletion) {
		return diag.Errorf("error updating Invitation %q: only %q attribute can be updated for Invitation", d.Id(), paramAllowDeletion)
	}
	return invitationRead(ctx, d, meta)
}

func invitationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting invitation %q", d.Id()), map[string]interface{}{invitationloggingKey: d.Id()})

	invitationId := d.Id()

	if invitationId == "" {
		return diag.Errorf("error deleting invitation: invitation id is missing")
	}

	c := meta.(*Client)

	if d.Get(paramAllowDeletion).(bool) == true && d.Get(paramStatus).(string) == statusAccepted {
		tflog.Debug(ctx, fmt.Sprintf("Deleted accepted Invitation %q from TF state since allow_deletion is set to true", invitationId), map[string]interface{}{invitationloggingKey: invitationId})
		return nil
	}

	req := c.iamClient.InvitationsIamV2Api.DeleteIamV2Invitation(c.iamApiContext(ctx), invitationId)
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting invitation %q: %s", invitationId, createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting invitation %q", invitationId), map[string]interface{}{invitationloggingKey: invitationId})

	return nil
}

func executeInvitationRead(ctx context.Context, c *Client, invitationId string) (iam.IamV2Invitation, *http.Response, error) {
	req := c.iamClient.InvitationsIamV2Api.GetIamV2Invitation(c.iamApiContext(ctx), invitationId)
	return req.Execute()
}

func executeInvitationCreate(ctx context.Context, c *Client, invitation *iam.IamV2Invitation) (iam.IamV2Invitation, *http.Response, error) {
	req := c.iamClient.InvitationsIamV2Api.CreateIamV2Invitation(c.iamApiContext(ctx)).IamV2Invitation(*invitation)
	return req.Execute()
}

func setInvitationAttributes(d *schema.ResourceData, invitation iam.IamV2Invitation) (*schema.ResourceData, error) {
	if err := d.Set(paramEmail, invitation.GetEmail()); err != nil {
		return nil, err
	}
	if err := d.Set(paramExpiresAt, invitation.GetExpiresAt().String()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAcceptedAt, invitation.GetAcceptedAt().String()); err != nil {
		return nil, err
	}
	if err := d.Set(paramStatus, invitation.GetStatus()); err != nil {
		return nil, err
	}
	if err := d.Set(paramAuthType, invitation.GetAuthType()); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramUser, paramId, invitation.GetUser().Id, d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramCreator, paramId, invitation.GetCreator().Id, d); err != nil {
		return nil, err
	}

	d.SetId(invitation.GetId())
	return d, nil
}
