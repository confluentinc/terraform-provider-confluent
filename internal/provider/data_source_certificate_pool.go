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

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func certificatePoolDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: certificatePoolDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Certificate Pool, for example, `op-abc123`.",
			},
			paramCertificateAuthority: certificateAuthorityDataSourceSchema(),
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A name for the Certificate Pool.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the Certificate Pool.",
			},
			paramExternalIdentifier: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The certificate field that will be used to represent the pool's external identity for audit logging.",
			},
			paramFilter: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A filter expression in Supported Common Expression Language (CEL) that specifies which identities can authenticate using your certificate pool.",
			},
		},
	}
}

func certificatePoolDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	certificatePoolId := d.Get(paramId).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Certificate Pool %q=%q", paramId, certificatePoolId), map[string]interface{}{certificatePoolKey: certificatePoolId})

	c := meta.(*Client)
	CertificateAuthorityId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)
	request := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPool(c.caApiContext(ctx), CertificateAuthorityId, certificatePoolId)
	certificatePool, resp, err := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPoolExecute(request)
	if err != nil {
		return diag.Errorf("error reading Certificate Pool %q: %s", certificatePoolId, createDescriptiveError(err, resp))
	}
	certificatePoolJson, err := json.Marshal(certificatePool)
	if err != nil {
		return diag.Errorf("error reading Certificate Pool %q: error marshaling %#v to json: %s", certificatePoolId, certificatePool, createDescriptiveError(err, resp))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Certificate Pool %q: %s", certificatePoolId, certificatePoolJson), map[string]interface{}{certificatePoolKey: certificatePoolId})

	if _, err := setCertificatePoolAttributes(d, certificatePool, CertificateAuthorityId); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func certificateAuthorityDataSourceSchema() *schema.Schema {
	return &schema.Schema{
		Type: schema.TypeList,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		Required: true,
		MaxItems: 1,
	}
}
