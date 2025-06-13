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
	parent "github.com/confluentinc/ccloud-sdk-go-v2-internal/parent/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/http"
)

const (
	paramParent = "parent"
)

func parentOrganizationLinkResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: parentOrganizationLinkCreate,
		ReadContext:   parentOrganizationLinkRead,
		DeleteContext: parentOrganizationLinkDelete,
		Importer: &schema.ResourceImporter{
			StateContext: parentOrganizationLinkImport,
		},
		Schema: map[string]*schema.Schema{
			paramParent:       environmentSchema(),
			paramOrganization: environmentSchema(),
		},
	}
}

func parentOrganizationLinkCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	parentID := extractStringValueFromBlock(d, paramParent, paramId)
	organizationID := extractStringValueFromBlock(d, paramOrganization, paramId)

	createParentOrganizationLinkRequest := parent.NewIamV2ParentOrganizationLink()
	createParentOrganizationLinkRequest.SetOrganizationId(organizationID)
	createParentOrganizationLinkRequest.SetParentId(parentID)

	createParentOrganizationLinkRequestJson, err := json.Marshal(createParentOrganizationLinkRequest)
	if err != nil {
		return diag.Errorf("error creating Parent Organization Link: error marshaling %#v to json: %s", createParentOrganizationLinkRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Parent Organization Link: %s", createParentOrganizationLinkRequestJson))

	createdParentOrganizationLink, _, err := executeParentOrganizationLinkCreate(c.parentApiContext(ctx), c, *createParentOrganizationLinkRequest)
	if err != nil {
		return diag.Errorf("error creating Parent Organization Link: %s", createDescriptiveError(err))
	}
	d.SetId(createdParentOrganizationLink.GetId())

	createdParentOrganizationLinkJson, err := json.Marshal(createdParentOrganizationLink)
	if err != nil {
		return diag.Errorf("error creating Parent Organization Link %q: error marshaling %#v to json: %s", d.Id(), createdParentOrganizationLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Parent Organization Link %q: %s", d.Id(), createdParentOrganizationLinkJson), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})

	return parentOrganizationLinkRead(ctx, d, meta)
}

func executeParentOrganizationLinkCreate(ctx context.Context, c *Client, parentOrganizationLink parent.IamV2ParentOrganizationLink) (parent.IamV2ParentOrganizationLink, *http.Response, error) {
	req := c.parentClient.ParentOrganizationLinksIamV2Api.CreateIamV2ParentOrganizationLink(c.parentApiContext(ctx)).IamV2ParentOrganizationLink(parentOrganizationLink)
	return req.Execute()
}

func executeParentOrganizationLinkRead(ctx context.Context, c *Client, parentOrganizationLinkId string) (parent.IamV2ParentOrganizationLink, *http.Response, error) {
	req := c.parentClient.ParentOrganizationLinksIamV2Api.GetIamV2ParentOrganizationLink(c.parentApiContext(ctx), parentOrganizationLinkId)
	return req.Execute()
}

func parentOrganizationLinkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})

	parentOrganizationLinkId := d.Id()

	if _, err := readParentOrganizationLinkAndSetAttributes(ctx, d, meta, parentOrganizationLinkId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Parent Organization Link %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readParentOrganizationLinkAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, parentOrganizationLinkId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	parentOrganizationLink, resp, err := executeParentOrganizationLinkRead(c.parentApiContext(ctx), c, parentOrganizationLinkId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Parent Organization Link %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Parent Organization Link %q in TF state because Parent Organization Link could not be found on the server", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	parentOrganizationLinkJson, err := json.Marshal(parentOrganizationLink)
	if err != nil {
		return nil, fmt.Errorf("error reading Parent Organization Link %q: error marshaling %#v to json: %s", parentOrganizationLinkId, parentOrganizationLink, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Parent Organization Link %q: %s", d.Id(), parentOrganizationLinkJson), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})

	if _, err := setParentOrganizationLinkAttributes(d, parentOrganizationLink); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setParentOrganizationLinkAttributes(d *schema.ResourceData, parentOrganizationLink parent.IamV2ParentOrganizationLink) (*schema.ResourceData, error) {
	if err := setStringAttributeInListBlockOfSizeOne(paramOrganization, paramId, parentOrganizationLink.GetOrganizationId(), d); err != nil {
		return nil, err
	}
	if err := setStringAttributeInListBlockOfSizeOne(paramParent, paramId, parentOrganizationLink.GetParentId(), d); err != nil {
		return nil, err
	}
	d.SetId(parentOrganizationLink.GetId())
	return d, nil
}

func parentOrganizationLinkDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.parentClient.ParentOrganizationLinksIamV2Api.DeleteIamV2ParentOrganizationLink(c.parentApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Parent Organization Link %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})

	return nil
}

func parentOrganizationLinkImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readParentOrganizationLinkAndSetAttributes(ctx, d, meta, d.Id()); err != nil {
		return nil, fmt.Errorf("error importing Parent Organization Link %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Parent Organization Link %q", d.Id()), map[string]interface{}{parentOrganizationLinkLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
