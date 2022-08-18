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
	oidc "github.com/confluentinc/ccloud-sdk-go-v2/identity-provider/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/http"
)

const (
	paramIssuer  = "issuer"
	paramJwksUri = "jwks_uri"
)

func identityProviderResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: identityProviderCreate,
		ReadContext:   identityProviderRead,
		UpdateContext: identityProviderUpdate,
		DeleteContext: identityProviderDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A name for the Identity Provider.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A description of the Identity Provider.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramIssuer: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "A publicly reachable issuer URI for the Identity Provider.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramJwksUri: {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "A publicly reachable JWKS URI for the Identity Provider.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func identityProviderUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription) {
		return diag.Errorf("error updating Identity Provider %q: only %q, %q attributes can be updated for Identity Provider", d.Id(), paramDisplayName, paramDescription)
	}

	updateIdentityProviderRequest := oidc.NewIamV2IdentityProviderUpdate()

	if d.HasChange(paramDisplayName) {
		updatedDisplayName := d.Get(paramDisplayName).(string)
		updateIdentityProviderRequest.SetDisplayName(updatedDisplayName)
	}
	if d.HasChange(paramDescription) {
		updatedDescription := d.Get(paramDescription).(string)
		updateIdentityProviderRequest.SetDescription(updatedDescription)
	}

	updateIdentityProviderRequestJson, err := json.Marshal(updateIdentityProviderRequest)
	if err != nil {
		return diag.Errorf("error updating Identity Provider %q: error marshaling %#v to json: %s", d.Id(), updateIdentityProviderRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Identity Provider %q: %s", d.Id(), updateIdentityProviderRequestJson), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedIdentityProvider, _, err := c.oidcClient.IdentityProvidersIamV2Api.UpdateIamV2IdentityProvider(c.oidcApiContext(ctx), d.Id()).IamV2IdentityProviderUpdate(*updateIdentityProviderRequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Identity Provider %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedIdentityProviderJson, err := json.Marshal(updatedIdentityProvider)
	if err != nil {
		return diag.Errorf("error updating Identity Provider %q: error marshaling %#v to json: %s", d.Id(), updatedIdentityProvider, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Identity Provider %q: %s", d.Id(), updatedIdentityProviderJson), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	return identityProviderRead(ctx, d, meta)
}

func identityProviderCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	displayName := d.Get(paramDisplayName).(string)
	description := d.Get(paramDescription).(string)
	issuer := d.Get(paramIssuer).(string)
	jwksUri := d.Get(paramJwksUri).(string)

	createIdentityProviderRequest := oidc.NewIamV2IdentityProvider()
	createIdentityProviderRequest.SetDisplayName(displayName)
	createIdentityProviderRequest.SetDescription(description)
	createIdentityProviderRequest.SetIssuer(issuer)
	createIdentityProviderRequest.SetJwksUri(jwksUri)
	createIdentityProviderRequestJson, err := json.Marshal(createIdentityProviderRequest)
	if err != nil {
		return diag.Errorf("error creating Identity Provider: error marshaling %#v to json: %s", createIdentityProviderRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Identity Provider: %s", createIdentityProviderRequestJson))

	createdIdentityProvider, _, err := executeIdentityProviderCreate(c.oidcApiContext(ctx), c, createIdentityProviderRequest)
	if err != nil {
		return diag.Errorf("error creating Identity Provider: %s", createDescriptiveError(err))
	}
	d.SetId(createdIdentityProvider.GetId())

	createdIdentityProviderJson, err := json.Marshal(createdIdentityProvider)
	if err != nil {
		return diag.Errorf("error creating Identity Provider: %q: error marshaling %#v to json: %s", createdIdentityProvider.GetId(), createdIdentityProvider, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Identity Provider %q: %s", d.Id(), createdIdentityProviderJson), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	return identityProviderRead(ctx, d, meta)
}

func executeIdentityProviderCreate(ctx context.Context, c *Client, identityProvider *oidc.IamV2IdentityProvider) (oidc.IamV2IdentityProvider, *http.Response, error) {
	req := c.oidcClient.IdentityProvidersIamV2Api.CreateIamV2IdentityProvider(c.oidcApiContext(ctx)).IamV2IdentityProvider(*identityProvider)
	return req.Execute()
}

func identityProviderDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Identity Provider %q", d.Id()), map[string]interface{}{identityProviderLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.oidcClient.IdentityProvidersIamV2Api.DeleteIamV2IdentityProvider(c.oidcApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Identity Provider %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Identity Provider %q", d.Id()), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	return nil
}

func identityProviderRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Identity Provider %q", d.Id()), map[string]interface{}{identityProviderLoggingKey: d.Id()})
	c := meta.(*Client)
	identityProvider, resp, err := executeIdentityProviderRead(c.oidcApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Identity Provider %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{identityProviderLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Identity Provider %q in TF state because Identity Provider could not be found on the server", d.Id()), map[string]interface{}{identityProviderLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	identityProviderJson, err := json.Marshal(identityProvider)
	if err != nil {
		return diag.Errorf("error reading Identity Provider %q: error marshaling %#v to json: %s", d.Id(), identityProvider, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Identity Provider %q: %s", d.Id(), identityProviderJson), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	if _, err := setIdentityProviderAttributes(d, identityProvider); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Identity Provider %q", d.Id()), map[string]interface{}{identityProviderLoggingKey: d.Id()})

	return nil
}
func executeIdentityProviderRead(ctx context.Context, c *Client, identityProviderId string) (oidc.IamV2IdentityProvider, *http.Response, error) {
	req := c.oidcClient.IdentityProvidersIamV2Api.GetIamV2IdentityProvider(c.oidcApiContext(ctx), identityProviderId)
	return req.Execute()
}

func setIdentityProviderAttributes(d *schema.ResourceData, identityProvider oidc.IamV2IdentityProvider) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, identityProvider.GetDisplayName()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramDescription, identityProvider.GetDescription()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramIssuer, identityProvider.GetIssuer()); err != nil {
		return nil, createDescriptiveError(err)
	}
	if err := d.Set(paramJwksUri, identityProvider.GetJwksUri()); err != nil {
		return nil, createDescriptiveError(err)
	}

	d.SetId(identityProvider.GetId())
	return d, nil
}
