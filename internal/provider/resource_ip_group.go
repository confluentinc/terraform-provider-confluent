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
	"net/http"

	iamip "github.com/confluentinc/ccloud-sdk-go-v2/iam-ip-filtering/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramGroupName  = "group_name"
	paramCidrBlocks = "cidr_blocks"
)

func ipGroupResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: ipGroupCreate,
		ReadContext:   ipGroupRead,
		UpdateContext: ipGroupUpdate,
		DeleteContext: ipGroupDelete,
		Importer: &schema.ResourceImporter{
			StateContext: ipGroupImport,
		},
		Schema: map[string]*schema.Schema{
			paramGroupName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human readable name for an IP Group.",
				ValidateFunc: validation.StringLenBetween(1, 64),
			},
			paramCidrBlocks: {
				Type:        schema.TypeSet,
				Description: "A list of CIDRs.",
				MinItems:    1,
				MaxItems:    25,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func ipGroupUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramGroupName, paramCidrBlocks) {
		return diag.Errorf("error updating IP Group %q: only %q and %q attributes can be updated for IP Group", d.Id(), paramGroupName, paramCidrBlocks)
	}

	// We need to construct a full object to avoid backend errors
	updateIPGroupRequest := buildIPGroupRequest(d)

	updateIPGroupRequestJson, err := json.Marshal(updateIPGroupRequest)
	if err != nil {
		return diag.Errorf("error updating IP Group %q: error marshaling %#v to json: %s", d.Id(), updateIPGroupRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating IP Group %q: %s", d.Id(), updateIPGroupRequestJson), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedIPGroup, _, err := c.iamIPClient.IPGroupsIamV2Api.UpdateIamV2IpGroup(c.iamIPApiContext(ctx), d.Id()).IamV2IpGroup(*updateIPGroupRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating IP Group %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedIPGroupJson, err := json.Marshal(updatedIPGroup)
	if err != nil {
		return diag.Errorf("error updating IP Group %q: error marshaling %#v to json: %s", d.Id(), updatedIPGroup, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating IP Group %q: %s", d.Id(), updatedIPGroupJson), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	return ipGroupRead(ctx, d, meta)
}

func ipGroupCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	createIPGroupRequest := buildIPGroupRequest(d)

	createIPGroupRequestJson, err := json.Marshal(createIPGroupRequest)
	if err != nil {
		return diag.Errorf("error creating IP Group: error marshaling %#v to json: %s", createIPGroupRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new IP Group: %s", createIPGroupRequestJson))

	createdIPGroup, resp, err := executeIPGroupCreate(c.iamIPApiContext(ctx), c, createIPGroupRequest)
	if err != nil {
		return diag.Errorf("error creating IP Group %q: %s", d.Get(paramGroupName), createDescriptiveError(err, resp))
	}
	d.SetId(createdIPGroup.GetId())

	createdIPGroupJson, err := json.Marshal(createdIPGroup)
	if err != nil {
		return diag.Errorf("error creating IP Group %q: error marshaling %#v to json: %s", d.Id(), createdIPGroup, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating IP Group %q: %s", d.Id(), createdIPGroupJson), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	return ipGroupRead(ctx, d, meta)
}

func executeIPGroupCreate(ctx context.Context, c *Client, ipGroup *iamip.IamV2IpGroup) (iamip.IamV2IpGroup, *http.Response, error) {
	req := c.iamIPClient.IPGroupsIamV2Api.CreateIamV2IpGroup(c.iamIPApiContext(ctx)).IamV2IpGroup(*ipGroup)
	return req.Execute()
}

func ipGroupDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.iamIPClient.IPGroupsIamV2Api.DeleteIamV2IpGroup(c.iamIPApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting IP Group %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	return nil
}

func executeIPGroupRead(ctx context.Context, c *Client, ipGroupId string) (iamip.IamV2IpGroup, *http.Response, error) {
	req := c.iamIPClient.IPGroupsIamV2Api.GetIamV2IpGroup(c.iamIPApiContext(ctx), ipGroupId)
	return req.Execute()
}

func ipGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})
	c := meta.(*Client)
	ipGroup, resp, err := executeIPGroupRead(c.iamIPApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading IP Group %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{ipGroupLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing IP Group %q in TF state because IP Group could not be found on the server", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	ipGroupJson, err := json.Marshal(ipGroup)
	if err != nil {
		return diag.Errorf("error reading IP Group %q: error marshaling %#v to json: %s", d.Id(), ipGroup, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched IP Group %q: %s", d.Id(), ipGroupJson), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	if _, err := setIPGroupAttributes(d, ipGroup); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})

	return nil
}

func ipGroupImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := ipGroupRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing IP Group %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing IP Group %q", d.Id()), map[string]interface{}{ipGroupLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setIPGroupAttributes(d *schema.ResourceData, ipGroup iamip.IamV2IpGroup) (*schema.ResourceData, error) {
	if err := d.Set(paramGroupName, ipGroup.GetGroupName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramCidrBlocks, ipGroup.GetCidrBlocks()); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(ipGroup.GetId())
	return d, nil
}

func buildIPGroupRequest(d *schema.ResourceData) *iamip.IamV2IpGroup {
	groupName := d.Get(paramGroupName).(string)
	cidrBlocks := convertToStringSlice(d.Get(paramCidrBlocks).(*schema.Set).List())

	req := iamip.NewIamV2IpGroup()
	req.SetGroupName(groupName)
	req.SetCidrBlocks(cidrBlocks)
	return req
}
