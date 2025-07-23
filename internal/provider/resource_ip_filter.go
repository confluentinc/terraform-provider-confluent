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
	paramFilterName      = "filter_name"
	paramResourceGroup   = "resource_group"
	paramResourceScope   = "resource_scope"
	paramOperationGroups = "operation_groups"
	paramIPGroups        = "ip_groups"
)

func ipFilterResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: ipFilterCreate,
		ReadContext:   ipFilterRead,
		UpdateContext: ipFilterUpdate,
		DeleteContext: ipFilterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: ipFilterImport,
		},
		Schema: map[string]*schema.Schema{
			paramFilterName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A human readable name for an IP Filter.",
				ValidateFunc: validation.StringLenBetween(1, 64),
			},
			paramResourceGroup: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Scope of resources covered by this IP filter.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramResourceScope: {
				Type:        schema.TypeString,
				Description: "A CRN that specifies the scope of the ip filter, specifically the organization or environment.",
				Optional:    true,
				Computed:    true,
			},
			paramOperationGroups: {
				Type:        schema.TypeSet,
				Description: "Scope of resources covered by this IP filter.",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			paramIPGroups: {
				Type:        schema.TypeSet,
				Description: "A list of IP Groups.",
				MinItems:    1,
				MaxItems:    25,
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func ipFilterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramFilterName, paramResourceGroup, paramOperationGroups, paramIPGroups) {
		return diag.Errorf("error updating IP Filter %q: only %q, %q, %q, and %q attributes can be updated for IP Filter", d.Id(), paramFilterName, paramResourceGroup, paramOperationGroups, paramIPGroups)
	}

	filterName := d.Get(paramFilterName).(string)
	resourceGroup := d.Get(paramResourceGroup).(string)
	resourceScope := d.Get(paramResourceScope).(string)
	operationGroups := convertToStringSlice(d.Get(paramOperationGroups).(*schema.Set).List())
	ipGroupStrings := convertToStringSlice(d.Get(paramIPGroups).(*schema.Set).List())

	updateIPFilterRequest := iamip.NewIamV2IpFilter()
	updateIPFilterRequest.SetFilterName(filterName)
	updateIPFilterRequest.SetResourceGroup(resourceGroup)
	if len(resourceScope) > 0 {
		updateIPFilterRequest.SetResourceScope(resourceScope)
	}
	if len(operationGroups) > 0 {
		updateIPFilterRequest.SetOperationGroups(operationGroups)
	}
	if len(ipGroupStrings) > 0 {
		var ipGroupRefs []iamip.GlobalObjectReference
		for _, id := range ipGroupStrings {
			ipGroupRefs = append(ipGroupRefs, iamip.GlobalObjectReference{Id: id})
		}

		updateIPFilterRequest.SetIpGroups(ipGroupRefs)
	}

	updateIPFilterRequestJson, err := json.Marshal(updateIPFilterRequest)
	if err != nil {
		return diag.Errorf("error updating IP Filter %q: error marshaling %#v to json: %s", d.Id(), updateIPFilterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating IP Filter %q: %s", d.Id(), updateIPFilterRequestJson), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedIPFilter, resp, err := c.iamIPClient.IPFiltersIamV2Api.UpdateIamV2IpFilter(c.iamIPApiContext(ctx), d.Id()).IamV2IpFilter(*updateIPFilterRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating IP Filter %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	updatedIPFilterJson, err := json.Marshal(updatedIPFilter)
	if err != nil {
		return diag.Errorf("error updating IP Filter %q: error marshaling %#v to json: %s", d.Id(), updatedIPFilter, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating IP Filter %q: %s", d.Id(), updatedIPFilterJson), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	return ipFilterRead(ctx, d, meta)
}

func ipFilterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	filterName := d.Get(paramFilterName).(string)
	resourceGroup := d.Get(paramResourceGroup).(string)
	resourceScope := d.Get(paramResourceScope).(string)
	operationGroups := convertToStringSlice(d.Get(paramOperationGroups).(*schema.Set).List())
	ipGroupStrings := convertToStringSlice(d.Get(paramIPGroups).(*schema.Set).List())

	createIPFilterRequest := iamip.NewIamV2IpFilter()
	createIPFilterRequest.SetFilterName(filterName)
	createIPFilterRequest.SetResourceGroup(resourceGroup)
	if len(resourceScope) > 0 {
		createIPFilterRequest.SetResourceScope(resourceScope)
	}
	if len(operationGroups) > 0 {
		createIPFilterRequest.SetOperationGroups(operationGroups)
	}
	if len(ipGroupStrings) > 0 {
		var ipGroupRefs []iamip.GlobalObjectReference
		for _, id := range ipGroupStrings {
			ipGroupRefs = append(ipGroupRefs, iamip.GlobalObjectReference{Id: id})
		}

		createIPFilterRequest.SetIpGroups(ipGroupRefs)
	}
	createIPFilterRequestJson, err := json.Marshal(createIPFilterRequest)
	if err != nil {
		return diag.Errorf("error creating IP Filter: error marshaling %#v to json: %s", createIPFilterRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new IP Filter: %s", createIPFilterRequestJson))

	createdIPFilter, resp, err := executeIPFilterCreate(c.iamIPApiContext(ctx), c, createIPFilterRequest)
	if err != nil {
		return diag.Errorf("error creating IP Filter %q: %s", filterName, createDescriptiveError(err, resp))
	}
	d.SetId(createdIPFilter.GetId())

	createdIPFilterJson, err := json.Marshal(createdIPFilter)
	if err != nil {
		return diag.Errorf("error creating IP Filter %q: error marshaling %#v to json: %s", d.Id(), createdIPFilter, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating IP Filter %q: %s", d.Id(), createdIPFilterJson), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	return ipFilterRead(ctx, d, meta)
}

func executeIPFilterCreate(ctx context.Context, c *Client, ipFilter *iamip.IamV2IpFilter) (iamip.IamV2IpFilter, *http.Response, error) {
	req := c.iamIPClient.IPFiltersIamV2Api.CreateIamV2IpFilter(c.iamIPApiContext(ctx)).IamV2IpFilter(*ipFilter)
	return req.Execute()
}

func ipFilterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.iamIPClient.IPFiltersIamV2Api.DeleteIamV2IpFilter(c.iamIPApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting IP Filter %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	return nil
}

func executeIPFilterRead(ctx context.Context, c *Client, ipFilterId string) (iamip.IamV2IpFilter, *http.Response, error) {
	req := c.iamIPClient.IPFiltersIamV2Api.GetIamV2IpFilter(c.iamIPApiContext(ctx), ipFilterId)
	return req.Execute()
}

func ipFilterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})
	c := meta.(*Client)
	ipFilter, resp, err := executeIPFilterRead(c.iamIPApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading IP Filter %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{ipFilterLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing IP Filter %q in TF state because IP Filter could not be found on the server", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	ipFilterJson, err := json.Marshal(ipFilter)
	if err != nil {
		return diag.Errorf("error reading IP Filter %q: error marshaling %#v to json: %s", d.Id(), ipFilter, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched IP Filter %q: %s", d.Id(), ipFilterJson), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	if _, err := setIPFilterAttributes(d, ipFilter); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})

	return nil
}

func ipFilterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := ipFilterRead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing IP Filter %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing IP Filter %q", d.Id()), map[string]interface{}{ipFilterLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func setIPFilterAttributes(d *schema.ResourceData, ipFilter iamip.IamV2IpFilter) (*schema.ResourceData, error) {
	if err := d.Set(paramFilterName, ipFilter.GetFilterName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceGroup, ipFilter.GetResourceGroup()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramResourceScope, ipFilter.GetResourceScope()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramOperationGroups, ipFilter.GetOperationGroups()); err != nil {
		return nil, createDescriptiveError(err)
	}

	var ipGroupIDs []string
	for _, ref := range ipFilter.GetIpGroups() {
		ipGroupIDs = append(ipGroupIDs, ref.Id)
	}
	if err := d.Set(paramIPGroups, ipGroupIDs); err != nil {
		return nil, createDescriptiveError(err)
	}

	d.SetId(ipFilter.GetId())
	return d, nil
}
