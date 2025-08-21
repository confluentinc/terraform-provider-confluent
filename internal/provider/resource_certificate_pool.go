// Copyright 2024 Confluent Inc. All Rights Reserved.
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
	"strings"

	ca "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramExternalIdentifier   = "external_identifier"
	paramCertificateAuthority = "certificate_authority"
)

func certificatePoolResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: certificatePoolCreate,
		ReadContext:   certificatePoolRead,
		UpdateContext: certificatePoolUpdate,
		DeleteContext: certificatePoolDelete,
		Importer: &schema.ResourceImporter{
			StateContext: certificatePoolImport,
		},
		Schema: map[string]*schema.Schema{
			paramCertificateAuthority: certificateAuthoritySchema(),
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A name for the Certificate Pool.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A description of the Certificate Pool.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramExternalIdentifier: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The certificate field that will be used to represent the pool's external identity for audit logging.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFilter: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A filter expression in Supported Common Expression Language (CEL) that specifies which identities can authenticate using your certificate pool.",
				ValidateFunc: validation.StringLenBetween(1, 300),
			},
		},
	}
}

func certificatePoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	certificateAuthorityId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)

	createCertificatePoolRequest := ca.NewIamV2CertificateIdentityPool()
	createCertificatePoolRequest.SetDisplayName(d.Get(paramDisplayName).(string))
	createCertificatePoolRequest.SetDescription(d.Get(paramDescription).(string))
	createCertificatePoolRequest.SetExternalIdentifier(d.Get(paramExternalIdentifier).(string))
	createCertificatePoolRequest.SetFilter(d.Get(paramFilter).(string))

	createCertificatePoolRequestJson, err := json.Marshal(createCertificatePoolRequest)
	if err != nil {
		return diag.Errorf("error creating Certificate Pool: error marshaling %#v to json: %s", createCertificatePoolRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Certificate Pool: %s", createCertificatePoolRequestJson))

	req := c.caClient.CertificateIdentityPoolsIamV2Api.CreateIamV2CertificateIdentityPool(c.caApiContext(ctx), certificateAuthorityId).IamV2CertificateIdentityPool(*createCertificatePoolRequest)
	createdCertificatePool, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Certificate Pool %q: %s", createdCertificatePool.GetId(), createDescriptiveError(err, resp))
	}
	d.SetId(createdCertificatePool.GetId())

	createdCertificatePoolJson, err := json.Marshal(createdCertificatePool)
	if err != nil {
		return diag.Errorf("error creating Certificate Pool %q: error marshaling %#v to json: %s", d.Id(), createdCertificatePool, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Certificate Pool %q: %s", d.Id(), createdCertificatePoolJson), map[string]interface{}{certificatePoolKey: d.Id()})

	return certificatePoolRead(ctx, d, meta)
}

func executecertificatePoolRead(ctx context.Context, c *Client, certificateAuthorityId, certificatePoolId string) (ca.IamV2CertificateIdentityPool, *http.Response, error) {
	req := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPool(c.caApiContext(ctx), certificateAuthorityId, certificatePoolId)
	return req.Execute()
}

func certificatePoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading CertificatePool %q", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})

	certificatePoolId := d.Id()
	certificateAuthorityId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)

	if _, err := readCertificatePoolAndSetAttributes(ctx, d, meta, certificateAuthorityId, certificatePoolId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Certificate Pool %q: %s", certificatePoolId, createDescriptiveError(err)))
	}

	return nil
}

func readCertificatePoolAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, certificateAuthorityId, certificatePoolId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	certificatePool, resp, err := executecertificatePoolRead(c.caApiContext(ctx), c, certificateAuthorityId, certificatePoolId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Certificate Pool %q: %s", certificatePoolId, createDescriptiveError(err)), map[string]interface{}{certificatePoolKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Certificate Pool %q in TF state because Certificate Pool could not be found on the server", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	certificatePoolJson, err := json.Marshal(certificatePool)
	if err != nil {
		return nil, fmt.Errorf("error reading Certificate Pool %q: error marshaling %#v to json: %s", certificatePoolId, certificatePool, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Certificate Pool %q: %s", d.Id(), certificatePoolJson), map[string]interface{}{certificatePoolKey: d.Id()})

	if _, err := setCertificatePoolAttributes(d, certificatePool, certificateAuthorityId); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Certificate Pool %q", certificatePoolId), map[string]interface{}{certificatePoolKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setCertificatePoolAttributes(d *schema.ResourceData, certificatePool ca.IamV2CertificateIdentityPool, certificateAuthorityId string) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, certificatePool.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, certificatePool.GetDescription()); err != nil {
		return nil, err
	}
	if err := d.Set(paramExternalIdentifier, certificatePool.GetExternalIdentifier()); err != nil {
		return nil, err
	}
	if err := d.Set(paramFilter, certificatePool.GetFilter()); err != nil {
		return nil, err
	}

	if err := setStringAttributeInListBlockOfSizeOne(paramCertificateAuthority, paramId, certificateAuthorityId, d); err != nil {
		return nil, createDescriptiveError(err)
	}
	d.SetId(certificatePool.GetId())
	return d, nil
}

func certificatePoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Certificate Pool %q", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})
	certificateAuthorityId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)
	c := meta.(*Client)

	req := c.caClient.CertificateIdentityPoolsIamV2Api.DeleteIamV2CertificateIdentityPool(c.caApiContext(ctx), certificateAuthorityId, d.Id())
	_, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Certificate Pool %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Certificate Pool %q", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})

	return nil
}

func certificatePoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramExternalIdentifier, paramFilter) {
		return diag.Errorf("error updating Certificate Pool %q: only %q, %q, %q, %q attributes can be updated for Certificate Pool", d.Id(), paramDisplayName, paramDescription, paramExternalIdentifier, paramFilter)
	}

	updateCertificatePool := ca.NewIamV2CertificateIdentityPool()

	updateCertificatePool.SetDisplayName(d.Get(paramDisplayName).(string))
	updateCertificatePool.SetDescription(d.Get(paramDescription).(string))
	updateCertificatePool.SetExternalIdentifier(d.Get(paramExternalIdentifier).(string))
	updateCertificatePool.SetFilter(d.Get(paramFilter).(string))

	updateCertificatePoolJson, err := json.Marshal(updateCertificatePool)
	if err != nil {
		return diag.Errorf("error updating Certificate Pool %q: error marshaling %#v to json: %s", d.Id(), updateCertificatePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Certificate Pool %q: %s", d.Id(), updateCertificatePoolJson), map[string]interface{}{certificatePoolKey: d.Id()})

	c := meta.(*Client)
	certificateAuthorityId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)
	req := c.caClient.CertificateIdentityPoolsIamV2Api.UpdateIamV2CertificateIdentityPool(c.caApiContext(ctx), certificateAuthorityId, d.Id()).IamV2CertificateIdentityPool(*updateCertificatePool)
	updatedCertificatePool, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Certificate Pool %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	UpdatedCertificatePoolJson, err := json.Marshal(updatedCertificatePool)
	if err != nil {
		return diag.Errorf("error updating Certificate Pool %q: error marshaling %#v to json: %s", d.Id(), updatedCertificatePool, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Certificate Pool %q: %s", d.Id(), UpdatedCertificatePoolJson), map[string]interface{}{certificatePoolKey: d.Id()})
	return certificatePoolRead(ctx, d, meta)
}

func certificatePoolImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Certificate Pool %q", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})

	providerIdAndCertificatePoolId := d.Id()
	parts := strings.Split(providerIdAndCertificatePoolId, "/")

	if len(parts) != 2 {
		return nil, fmt.Errorf("error importing Certificate Pool: invalid format: expected '<provider ID>/<certificate pool ID>'")
	}

	providerId := parts[0]
	certificatePoolId := parts[1]
	d.SetId(certificatePoolId)

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readCertificatePoolAndSetAttributes(ctx, d, meta, providerId, d.Id()); err != nil {
		return nil, fmt.Errorf("error importing Certificate Pool %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Certificate Pool %q", d.Id()), map[string]interface{}{certificatePoolKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func certificateAuthoritySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		MinItems: 1,
		MaxItems: 1,
		Required: true,
		ForceNew: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:        schema.TypeString,
					Required:    true,
					ForceNew:    true,
					Description: "The unique identifier for the Certificate Authority.",
				},
			},
		},
	}
}
