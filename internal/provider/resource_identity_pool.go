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
	oidc "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
	"strings"
)

const (
	paramIdentityProvider = "identity_provider"
	paramIdentityClaim    = "identity_claim"
	paramFilter           = "filter"
)

func identityPoolResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: identityPoolCreate,
		ReadContext:   identityPoolRead,
		UpdateContext: identityPoolUpdate,
		DeleteContext: identityPoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: identityPoolImport,
		},
		Schema: map[string]*schema.Schema{
			paramIdentityProvider: identityProviderSchema(),
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A name for the Identity Pool.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A description of the Identity Pool.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramIdentityClaim: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A JWT claim to extract the authenticating principal to Confluent resources.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFilter: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A filter expression that must be evaluated to be true to use this identity pool.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func identityPoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramIdentityClaim, paramFilter) {
		return diag.Errorf("error updating Identity Pool %q: only %q, %q, %q, %q attributes can be updated for Identity Pool", d.Id(), paramDisplayName, paramDescription, paramIdentityClaim, paramFilter)
	}

	updateIdentityPoolRequest := oidc.NewIamV2IdentityPool()

	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateIdentityPoolRequest.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updateIdentityPoolRequest.SetDescription(updatedDescription)
	}
	if d.HasChange(paramIdentityClaim) {
		updatedIdentityClaim := d.Get(paramIdentityClaim).(string)
		updateIdentityPoolRequest.SetIdentityClaim(updatedIdentityClaim)
	}
	if d.HasChange(paramFilter) {
		updatedFilter := d.Get(paramFilter).(string)
		updateIdentityPoolRequest.SetFilter(updatedFilter)
	}

	updateIdentityPoolRequestJson, err := json.Marshal(updateIdentityPoolRequest)
	if err != nil {
		return diag.Errorf("error updating Identity Pool %q: error marshaling %#v to json: %s", d.Id(), updateIdentityPoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Identity Pool %q: %s", d.Id(), updateIdentityPoolRequestJson), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	c := meta.(*Client)
	identityProviderId := extractStringValueFromBlock(d, paramIdentityProvider, paramId)
	updatedIdentityPool, _, err := c.oidcClient.IdentityPoolsIamV2Api.UpdateIamV2IdentityPool(c.oidcApiContext(ctx), identityProviderId, d.Id()).IamV2IdentityPool(*updateIdentityPoolRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Identity Pool %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedIdentityPoolJson, err := json.Marshal(updatedIdentityPool)
	if err != nil {
		return diag.Errorf("error updating Identity Pool %q: error marshaling %#v to json: %s", d.Id(), updatedIdentityPool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Identity Pool %q: %s", d.Id(), updatedIdentityPoolJson), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	return identityPoolRead(ctx, d, meta)
}

func identityPoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	identityProviderId := extractStringValueFromBlock(d, paramIdentityProvider, paramId)
	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	identityClaim := d.Get(paramIdentityClaim).(string)
	filter := d.Get(paramFilter).(string)

	createIdentityPoolRequest := oidc.NewIamV2IdentityPool()
	createIdentityPoolRequest.SetDisplayName(displayName)
	createIdentityPoolRequest.SetDescription(description)
	createIdentityPoolRequest.SetIdentityClaim(identityClaim)
	createIdentityPoolRequest.SetFilter(filter)
	createIdentityPoolRequestJson, err := json.Marshal(createIdentityPoolRequest)
	if err != nil {
		return diag.Errorf("error creating Identity Pool: error marshaling %#v to json: %s", createIdentityPoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Identity Pool: %s", createIdentityPoolRequestJson))

	createdIdentityPool, _, err := executeIdentityPoolCreate(c.oidcApiContext(ctx), c, createIdentityPoolRequest, identityProviderId)
	if err != nil {
		return diag.Errorf("error creating Identity Pool: %s", createDescriptiveError(err))
	}
	d.SetId(createdIdentityPool.GetId())

	createdIdentityPoolJson, err := json.Marshal(createdIdentityPool)
	if err != nil {
		return diag.Errorf("error creating Identity Pool: %q: error marshaling %#v to json: %s", createdIdentityPool.GetId(), createdIdentityPool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Identity Pool %q: %s", d.Id(), createdIdentityPoolJson), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	return identityPoolRead(ctx, d, meta)
}

func executeIdentityPoolCreate(ctx context.Context, c *Client, identityPool *oidc.IamV2IdentityPool, identityProviderId string) (oidc.IamV2IdentityPool, *http.Response, error) {
	req := c.oidcClient.IdentityPoolsIamV2Api.CreateIamV2IdentityPool(c.oidcApiContext(ctx), identityProviderId).IamV2IdentityPool(*identityPool)
	return req.Execute()
}

func identityPoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})
	identityProviderId := extractStringValueFromBlock(d, paramIdentityProvider, paramId)
	c := meta.(*Client)

	req := c.oidcClient.IdentityPoolsIamV2Api.DeleteIamV2IdentityPool(c.oidcApiContext(ctx), identityProviderId, d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Identity Pool %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	return nil
}

func identityPoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	identityPoolId := d.Id()
	identityProviderId := extractStringValueFromBlock(d, paramIdentityProvider, paramId)

	if _, err := readIdentityPoolAndSetAttributes(ctx, d, meta, identityProviderId, identityPoolId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Identity Pool %q: %s", d.Id(), createDescriptiveError(err)))
	}

	return nil
}

func readIdentityPoolAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, identityProviderId, identityPoolId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	identityPool, resp, err := executeIdentityPoolRead(c.oidcApiContext(ctx), c, d.Id(), identityProviderId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Identity Pool %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{identityPoolLoggingKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Identity Pool %q in TF state because Identity Pool could not be found on the server", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, createDescriptiveError(err)
	}
	identityPoolJson, err := json.Marshal(identityPool)
	if err != nil {
		return nil, fmt.Errorf("error reading Identity Pool %q: error marshaling %#v to json: %s", d.Id(), identityPool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Identity Pool %q: %s", d.Id(), identityPoolJson), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	if _, err := setIdentityPoolAttributes(d, identityPool, identityProviderId); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func executeIdentityPoolRead(ctx context.Context, c *Client, identityPoolId, identityProviderId string) (oidc.IamV2IdentityPool, *http.Response, error) {
	req := c.oidcClient.IdentityPoolsIamV2Api.GetIamV2IdentityPool(c.oidcApiContext(ctx), identityProviderId, identityPoolId)
	return req.Execute()
}

func setIdentityPoolAttributes(d *schema.ResourceData, identityPool oidc.IamV2IdentityPool, identityProviderId string) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, identityPool.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, identityPool.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramIdentityClaim, identityPool.GetIdentityClaim()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramFilter, identityPool.GetFilter()); err != nil {
		return nil, createDescriptiveError(err)
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramIdentityProvider, paramId, identityProviderId, d); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(identityPool.GetId())
	return d, nil
}

func identityPoolImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})

	identityProviderIdAndIdentityPoolId := d.Id()
	parts := strings.Split(identityProviderIdAndIdentityPoolId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Identity Pool: invalid format: expected '<identity provider ID>/<identity pool ID>'")
	}

	identityProviderId := parts[0]
	identityPoolId := parts[1]
	d.SetId(identityPoolId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readIdentityPoolAndSetAttributes(ctx, d, meta, identityProviderId, identityPoolId); err != nil {
		return nil, fmt.Errorf("error importing Identity Pool %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Identity Pool %q", d.Id()), map[string]interface{}{identityPoolLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

// https://github.com/hashicorp/terraform-plugin-sdk/issues/155#issuecomment-489699737
////  alternative - https://github.com/hashicorp/terraform-plugin-sdk/issues/248#issuecomment-725013327
func identityProviderSchema() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		MinItems:    1,
		MaxItems:    1,
		Required:    true,
		ForceNew:    true,
		Description: "Identity Provider objects represent external OAuth/OpenID Connect providers within Confluent Cloud.",
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the Identity Provider.",
				},
			},
		},
	}
}
