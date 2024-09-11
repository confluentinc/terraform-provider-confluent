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

func certificateAuthorityDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: certificateAuthorityDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The ID of the Certificate Authority, for example, `op-abc123`.",
			},
			paramDisplayName: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A name for the Certificate Authority.",
			},
			paramDescription: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "A description of the Certificate Authority.",
			},
			paramCertificateChainFilename: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the certificate file.",
			},
			paramFingerprints: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The fingerprints for each certificate in the certificate chain.",
			},
			paramSerialNumbers: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The serial numbers for each certificate in the certificate chain.",
			},
			paramCrlSource: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			paramCrlUrl: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The url from which to fetch the CRL for the certificate authority if crl_source is URL.",
			},
		},
	}
}

func certificateAuthorityDataSourceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	certificateAuthorityId := d.Get(paramId).(string)

	tflog.Debug(ctx, fmt.Sprintf("Reading Certificate Authority %q=%q", paramId, certificateAuthorityId), map[string]interface{}{certificateAuthorityKey: certificateAuthorityId})

	c := meta.(*Client)
	request := c.caClient.CertificateAuthoritiesIamV2Api.GetIamV2CertificateAuthority(c.caApiContext(ctx), certificateAuthorityId)
	certificateAuthority, _, err := c.caClient.CertificateAuthoritiesIamV2Api.GetIamV2CertificateAuthorityExecute(request)
	if err != nil {
		return diag.Errorf("error reading Certificate Authority %q: %s", certificateAuthorityId, createDescriptiveError(err))
	}
	certificateAuthorityJson, err := json.Marshal(certificateAuthority)
	if err != nil {
		return diag.Errorf("error reading Certificate Authority %q: error marshaling %#v to json: %s", certificateAuthorityId, certificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Certificate Authority %q: %s", certificateAuthorityId, certificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: certificateAuthorityId})

	if _, err := setCertificateAuthorityAttributes(d, certificateAuthority); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}
