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
	"net/http"

	parent "github.com/confluentinc/ccloud-sdk-go-v2-internal/parent/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramConnectionName        = "connection_name"
	paramEmailAttributeMapping = "email_attribute_mapping"
	paramIdpInitiated          = "idp_initiated"
	paramJitEnabled            = "jit_enabled"
	paramBupEnabled            = "bup_enabled"
	paramSignInEndpoint        = "sign_in_endpoint"
	paramSigningCert           = "signing_cert"
	parentSSOLoggingKey        = "parent_sso"
)

func parentSSOResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: parentSSOCreate,
		ReadContext:   parentSSORead,
		UpdateContext: parentSSOUpdate,
		DeleteContext: parentSSODelete,
		Importer: &schema.ResourceImporter{
			StateContext: parentSSOImport,
		},
		Schema: map[string]*schema.Schema{
			paramConnectionName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Name to be used for the SSO connection.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramEmailAttributeMapping: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Name of the SAML attribute used to pass email in the SAMLResponse.",
			},
			paramIdpInitiated: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enables/disables IDP initiated SSO.",
			},
			paramJitEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable Just-In-Time user provisioning.",
			},
			paramBupEnabled: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable Bulk User Provisioning.",
			},
			paramSignInEndpoint: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "IDP hosted endpoint where the SAMLRequest will be sent.",
				ValidateFunc: validation.IsURLWithHTTPorHTTPS,
			},
			paramSigningCert: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "X.509 certificate for SAML signature verification.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramParent: optionalIdBlockSchema(),
		},
	}
}

func parentSSOUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramEmailAttributeMapping, paramIdpInitiated, paramJitEnabled, paramBupEnabled, paramSignInEndpoint, paramSigningCert) {
		return diag.Errorf("error updating Parent SSO %q: only %q, %q, %q, %q, %q, %q, and %q attributes can be updated for Parent SSO", d.Id(), paramConnectionName, paramEmailAttributeMapping, paramIdpInitiated, paramJitEnabled, paramBupEnabled, paramSignInEndpoint, paramSigningCert)
	}

	updateParentSSORequest := parent.NewIamV2ParentSSOUpdate()
	updateParentSSORequest.SetId(d.Id())

	if d.HasChange(paramEmailAttributeMapping) {
		emailAttributeMapping := d.Get(paramEmailAttributeMapping).(string)
		updateParentSSORequest.SetEmailAttributeMapping(emailAttributeMapping)
	}

	if d.HasChange(paramIdpInitiated) {
		idpInitiated := d.Get(paramIdpInitiated).(bool)
		updateParentSSORequest.SetIdpInitiated(idpInitiated)
	}

	if d.HasChange(paramJitEnabled) {
		jitEnabled := d.Get(paramJitEnabled).(bool)
		updateParentSSORequest.SetJitEnabled(jitEnabled)
	}

	if d.HasChange(paramBupEnabled) {
		bupEnabled := d.Get(paramBupEnabled).(bool)
		updateParentSSORequest.SetBupEnabled(bupEnabled)
	}

	if d.HasChange(paramSignInEndpoint) {
		signInEndpoint := d.Get(paramSignInEndpoint).(string)
		updateParentSSORequest.SetSignInEndpoint(signInEndpoint)
	}

	if d.HasChange(paramSigningCert) {
		signingCert := d.Get(paramSigningCert).(string)
		updateParentSSORequest.SetSigningCert(signingCert)
	}

	updateParentSSORequestJson, err := json.Marshal(updateParentSSORequest)
	if err != nil {
		return diag.Errorf("error updating Parent SSO %q: error marshaling %#v to json: %s", d.Id(), updateParentSSORequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Parent SSO %q: %s", d.Id(), updateParentSSORequestJson), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	c := meta.(*Client)
	updatedParentSSO, _, err := c.parentClient.ParentSSOsIamV2Api.UpdateIamV2ParentSSO(c.parentApiContext(ctx), d.Id()).IamV2ParentSSOUpdate(*updateParentSSORequest).Execute()

	if err != nil {
		return diag.Errorf("error updating Parent SSO %q: %s", d.Id(), createDescriptiveError(err))
	}

	updatedParentSSOJson, err := json.Marshal(updatedParentSSO)
	if err != nil {
		return diag.Errorf("error updating Parent SSO %q: error marshaling %#v to json: %s", d.Id(), updatedParentSSO, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Parent SSO %q: %s", d.Id(), updatedParentSSOJson), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	return parentSSORead(ctx, d, meta)
}

func parentSSOCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	connectionName := d.Get(paramConnectionName).(string)
	emailAttributeMapping := d.Get(paramEmailAttributeMapping).(string)
	idpInitiated := d.Get(paramIdpInitiated).(bool)
	jitEnabled := d.Get(paramJitEnabled).(bool)
	bupEnabled := d.Get(paramBupEnabled).(bool)
	signInEndpoint := d.Get(paramSignInEndpoint).(string)
	signingCert := d.Get(paramSigningCert).(string)

	createParentSSORequest := parent.NewIamV2ParentSSO()
	createParentSSORequest.SetConnectionName(connectionName)
	createParentSSORequest.SetEmailAttributeMapping(emailAttributeMapping)
	createParentSSORequest.SetIdpInitiated(idpInitiated)
	createParentSSORequest.SetJitEnabled(jitEnabled)
	createParentSSORequest.SetBupEnabled(bupEnabled)
	createParentSSORequest.SetSignInEndpoint(signInEndpoint)
	createParentSSORequest.SetSigningCert(signingCert)

	createParentSSORequestJson, err := json.Marshal(createParentSSORequest)
	if err != nil {
		return diag.Errorf("error creating Parent SSO: error marshaling %#v to json: %s", createParentSSORequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Parent SSO: %s", createParentSSORequestJson))

	createdParentSSO, _, err := executeParentSSOCreate(c.parentApiContext(ctx), c, createParentSSORequest)
	if err != nil {
		return diag.Errorf("error creating Parent SSO %q: %s", connectionName, createDescriptiveError(err))
	}
	d.SetId(createdParentSSO.GetId())

	createdParentSSOJson, err := json.Marshal(createdParentSSO)
	if err != nil {
		return diag.Errorf("error creating Parent SSO %q: error marshaling %#v to json: %s", d.Id(), createdParentSSO, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Parent SSO %q: %s", d.Id(), createdParentSSOJson), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	return parentSSORead(ctx, d, meta)
}

func executeParentSSOCreate(ctx context.Context, c *Client, parentSSO *parent.IamV2ParentSSO) (parent.IamV2ParentSSO, *http.Response, error) {
	req := c.parentClient.ParentSSOsIamV2Api.CreateIamV2ParentSSO(c.parentApiContext(ctx)).IamV2ParentSSO(*parentSSO)
	return req.Execute()
}

func parentSSODelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})
	c := meta.(*Client)

	req := c.parentClient.ParentSSOsIamV2Api.DeleteIamV2ParentSSO(c.parentApiContext(ctx), d.Id())
	_, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Parent SSO %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	return nil
}

func executeParentSSORead(ctx context.Context, c *Client, parentSSOId string) (parent.IamV2ParentSSO, *http.Response, error) {
	req := c.parentClient.ParentSSOsIamV2Api.GetIamV2ParentSSO(c.parentApiContext(ctx), parentSSOId)
	return req.Execute()
}

func parentSSORead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})
	c := meta.(*Client)
	parentSSO, resp, err := executeParentSSORead(c.parentApiContext(ctx), c, d.Id())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Parent SSO %q: %s", d.Id(), createDescriptiveError(err)), map[string]interface{}{parentSSOLoggingKey: d.Id()})

		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Parent SSO %q in TF state because Parent SSO could not be found on the server", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})
			d.SetId("")
			return nil
		}

		return diag.FromErr(createDescriptiveError(err))
	}
	parentSSOJson, err := json.Marshal(parentSSO)
	if err != nil {
		return diag.Errorf("error reading Parent SSO %q: error marshaling %#v to json: %s", d.Id(), parentSSO, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Parent SSO %q: %s", d.Id(), parentSSOJson), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	if _, err := setParentSSOAttributes(d, parentSSO); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})

	return nil
}

func setParentSSOAttributes(d *schema.ResourceData, parentSSO parent.IamV2ParentSSO) (*schema.ResourceData, error) {
	if err := d.Set(paramConnectionName, parentSSO.GetConnectionName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramEmailAttributeMapping, parentSSO.GetEmailAttributeMapping()); err != nil {
		return nil, err
	}
	if err := d.Set(paramIdpInitiated, parentSSO.GetIdpInitiated()); err != nil {
		return nil, err
	}
	if err := d.Set(paramJitEnabled, parentSSO.GetJitEnabled()); err != nil {
		return nil, err
	}
	if err := d.Set(paramBupEnabled, parentSSO.GetBupEnabled()); err != nil {
		return nil, err
	}
	if err := d.Set(paramSignInEndpoint, parentSSO.GetSignInEndpoint()); err != nil {
		return nil, err
	}
	// Note: signing_cert is not set here as it's not returned by the API for security reasons
	// The value remains in Terraform state from the original configuration
	if err := setStringAttributeInListBlockOfSizeOne(paramParent, paramId, parentSSO.GetParentId(), d); err != nil {
		return nil, err
	}
	d.SetId(parentSSO.GetId())
	return d, nil
}

func parentSSOImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})
	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if diagnostics := parentSSORead(ctx, d, meta); diagnostics != nil {
		return nil, fmt.Errorf("error importing Parent SSO %q: %s", d.Id(), diagnostics[0].Summary)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Parent SSO %q", d.Id()), map[string]interface{}{parentSSOLoggingKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}
