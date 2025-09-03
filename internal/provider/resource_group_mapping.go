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
	sso "github.com/confluentinc/ccloud-sdk-go-v2/sso/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

func groupMappingResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: groupMappingCreate,
		ReadContext:   groupMappingRead,
		UpdateContext: groupMappingUpdate,
		DeleteContext: groupMappingDelete,
		Importer: &schema.ResourceImporter{
			StateContext: groupMappingImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFilter: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human-readable name for the Group Mapping.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A free-form description of the Group Mapping.",
			},
		},
	}
}

func groupMappingUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramFilter, paramDisplayName, paramDescription) {
		return diag.Errorf("error updating Group Mapping %q: only %q, %q, %q attributes can be updated for Group Mapping", d.Id(), paramFilter, paramDisplayName, paramDescription)
	}

	updateGroupMappingRequest := sso.NewIamV2SsoGroupMapping()

	if d.HasChange(paramFilter) {
		updatedFilter := d.Get(paramFilter).(string)
		updateGroupMappingRequest.SetFilter(updatedFilter)
	}
	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateGroupMappingRequest.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updateGroupMappingRequest.SetDescription(updatedDescription)
	}

	updateGroupMappingRequestJson, err := json.Marshal(updateGroupMappingRequest)
	if err != nil {
		return diag.Errorf("error updating Group Mapping %q: error marshaling %#v to json: %s", d.Id(), updateGroupMappingRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Group Mapping %q: %s", d.Id(), updateGroupMappingRequestJson), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedGroupMapping, resp, err := c.ssoClient.GroupMappingsIamV2SsoApi.UpdateIamV2SsoGroupMapping(c.ssoApiContext(ctx), d.Id()).IamV2SsoGroupMapping(*updateGroupMappingRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Group Mapping %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedGroupMappingJson, err := json.Marshal(updatedGroupMapping)
	if err != nil {
		return diag.Errorf("error updating Group Mapping %q: error marshaling %#v to json: %s", d.Id(), updatedGroupMapping, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Group Mapping %q: %s", d.Id(), updatedGroupMappingJson), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	return groupMappingRead(ctx, d, meta)
}

func groupMappingCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	filter := d.Get(paramFilter).(string)
	description := d.Get(paramDescription).(string)

	createGroupMappingRequest := sso.NewIamV2SsoGroupMapping()
	createGroupMappingRequest.SetDisplayName(displayName)
	createGroupMappingRequest.SetFilter(filter)
	createGroupMappingRequest.SetDescription(description)
	createGroupMappingRequestJson, err := json.Marshal(createGroupMappingRequest)
	if err != nil {
		return diag.Errorf("error creating Group Mapping: error marshaling %#v to json: %s", createGroupMappingRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Group Mapping: %s", createGroupMappingRequestJson))

	createdGroupMapping, resp, err := executeGroupMappingCreate(c.ssoApiContext(ctx), c, createGroupMappingRequest)
	if err != nil {
		return diag.Errorf("error creating Group Mapping %q: %s", displayName, createDescriptiveError(err, resp))
	}
	d.SetId(createdGroupMapping.GetId())

	createdGroupMappingJson, err := json.Marshal(createdGroupMapping)
	if err != nil {
		return diag.Errorf("error creating Group Mapping %q: error marshaling %#v to json: %s", d.Id(), createdGroupMapping, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Group Mapping %q: %s", d.Id(), createdGroupMappingJson), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	return groupMappingRead(ctx, d, meta)
}

func executeGroupMappingCreate(ctx context.Context, c *Client, groupMapping *sso.IamV2SsoGroupMapping) (sso.IamV2SsoGroupMapping, *http.Response, error) {
	req := c.ssoClient.GroupMappingsIamV2SsoApi.CreateIamV2SsoGroupMapping(c.ssoApiContext(ctx)).IamV2SsoGroupMapping(*groupMapping)
	return req.Execute()
}

func groupMappingDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.ssoClient.GroupMappingsIamV2SsoApi.DeleteIamV2SsoGroupMapping(c.ssoApiContext(ctx), d.Id())
	resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Group Mapping %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	return nil
}

func executeGroupMappingRead(ctx context.Context, c *Client, groupMappingId string) (sso.IamV2SsoGroupMapping, *http.Response, error) {
	req := c.ssoClient.GroupMappingsIamV2SsoApi.GetIamV2SsoGroupMapping(c.ssoApiContext(ctx), groupMappingId)
	return req.Execute()
}

func groupMappingRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})
	c := meta.(*Client)
	groupMapping, resp, err := executeGroupMappingRead(c.ssoApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Group Mapping %q: %s", d.Id(), createDescriptiveError(err, resp)), map[string]interface{}{groupMappingLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Group Mapping %q in TF state because Group Mapping could not be found on the server", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err, resp))
	}
	groupMappingJson, err := json.Marshal(groupMapping)
	if err != nil {
		return diag.Errorf("error reading Group Mapping %q: error marshaling %#v to json: %s", d.Id(), groupMapping, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Group Mapping %q: %s", d.Id(), groupMappingJson), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	if _, err := setGroupMappingAttributes(d, groupMapping); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})

	return nil
}

func groupMappingImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := groupMappingRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Group Mapping %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Group Mapping %q", d.Id()), map[string]interface{}{groupMappingLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setGroupMappingAttributes(d *schema.ResourceData, groupMapping sso.IamV2SsoGroupMapping) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, groupMapping.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramFilter, groupMapping.GetFilter()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, groupMapping.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(groupMapping.GetId())
	return d, nil
}
