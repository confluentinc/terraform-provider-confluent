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
	ca "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func certificatePoolDataSource() *schema.Resource {
	return &schema.Resource{
		ReadContext: certificatePoolDataSourceRead,
		Schema: map[string]*schema.Schema{
			paramId: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The ID of the Certificate Pool, for example, `op-abc123`.",
				ExactlyOneOf: []string{paramId, paramDisplayName},
			},
			paramCertificateAuthority: certificateAuthorityDataSourceSchema(),
			paramDisplayName: {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "A name for the Certificate Pool.",
				ExactlyOneOf: []string{paramId, paramDisplayName},
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
	displayName := d.Get(paramDisplayName).(string)

	certificateProviderId := extractStringValueFromBlock(d, paramCertificateAuthority, paramId)

	if certificatePoolId != "" {
		return certificatePoolDataSourceReadUsingId(ctx, d, meta, certificateProviderId, certificatePoolId)
	} else if displayName != "" {
		return certificatePoolDataSourceReadUsingDisplayName(ctx, d, meta, certificateProviderId, displayName)
	} else {
		return diag.Errorf("error reading Certificate Pool: exactly one of %q or %q must be specified but they're both empty", paramId, paramDisplayName)
	}
}

func certificatePoolDataSourceReadUsingId(ctx context.Context, d *schema.ResourceData, meta interface{}, certificateProviderId, certificatePoolId string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Certificate Pool %q=%q", paramId, certificatePoolId), map[string]interface{}{certificatePoolKey: certificatePoolId})

	c := meta.(*Client)
	request := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPool(c.caApiContext(ctx), certificateProviderId, certificatePoolId)
	certificatePool, resp, err := c.caClient.CertificateIdentityPoolsIamV2Api.GetIamV2CertificateIdentityPoolExecute(request)
	if err != nil {
		return diag.Errorf("error reading Certificate Pool %q: %s", certificatePoolId, createDescriptiveError(err, resp))
	}
	certificatePoolJson, err := json.Marshal(certificatePool)
	if err != nil {
		return diag.Errorf("error reading Certificate Pool %q: error marshaling %#v to json: %s", certificatePoolId, certificatePool, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Certificate Pool %q: %s", certificatePoolId, certificatePoolJson), map[string]interface{}{certificatePoolKey: certificatePoolId})

	if _, err := setCertificatePoolAttributes(d, certificatePool, certificateProviderId); err != nil {
		return diag.FromErr(createDescriptiveError(err))
	}
	return nil
}

func certificatePoolDataSourceReadUsingDisplayName(ctx context.Context, d *schema.ResourceData, meta interface{}, certificateProviderId, displayName string) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading Certitificate Pool %q=%q", paramDisplayName, displayName))

	c := meta.(*Client)
	certificatePools, err := loadCertificatePools(ctx, c, certificateProviderId)
	if err != nil {
		return diag.Errorf("error reading Certificate Pool %q: %s", displayName, createDescriptiveError(err))
	}
	if orgHasMultipleCertificatePoolsWithTargetDisplayName(certificatePools, displayName) {
		return diag.Errorf("error reading Certificate Pool: there are multiple Certificate Pools with %q=%q", paramDisplayName, displayName)
	}

	for _, certificatePool := range certificatePools {
		if certificatePool.GetDisplayName() == displayName {
			if _, err := setCertificatePoolAttributes(d, certificatePool, certificateProviderId); err != nil {
				return diag.FromErr(createDescriptiveError(err))
			}
			return nil
		}
	}

	return diag.Errorf("error reading Certificate Pool: Certificate Pool with %q=%q was not found", paramDisplayName, displayName)
}

func orgHasMultipleCertificatePoolsWithTargetDisplayName(certificatePools []ca.IamV2CertificateIdentityPool, displayName string) bool {
	var numberOfCertificatePoolsWithTargetDisplayName = 0
	for _, certificatePool := range certificatePools {
		if certificatePool.GetDisplayName() == displayName {
			numberOfCertificatePoolsWithTargetDisplayName += 1
		}
	}
	return numberOfCertificatePoolsWithTargetDisplayName > 1
}

func loadCertificatePools(ctx context.Context, c *Client, certificateProviderId string) ([]ca.IamV2CertificateIdentityPool, error) {
	certificatePools := make([]ca.IamV2CertificateIdentityPool, 0)

	allCertificatePoolsAreCollected := false
	pageToken := ""
	for !allCertificatePoolsAreCollected {
		certificatePoolsPageList, resp, err := executeListCertificatePools(ctx, c, certificateProviderId, pageToken)
		if err != nil {
			return nil, fmt.Errorf("error reading Certificate Pools: %s", createDescriptiveError(err, resp))
		}
		certificatePools = append(certificatePools, certificatePoolsPageList.GetData()...)

		// nextPageUrlStringNullable is nil for the last page
		nextPageUrlStringNullable := certificatePoolsPageList.GetMetadata().Next

		if nextPageUrlStringNullable.IsSet() {
			nextPageUrlString := *nextPageUrlStringNullable.Get()
			if nextPageUrlString == "" {
				allCertificatePoolsAreCollected = true
			} else {
				pageToken, err = extractPageToken(nextPageUrlString)
				if err != nil {
					return nil, fmt.Errorf("error reading Certificate Pools: %s", createDescriptiveError(err, resp))
				}
			}
		} else {
			allCertificatePoolsAreCollected = true
		}
	}
	return certificatePools, nil
}

func executeListCertificatePools(ctx context.Context, c *Client, certificateProviderId, pageToken string) (ca.IamV2CertificateIdentityPoolList, *http.Response, error) {
	if pageToken != "" {
		return c.caClient.CertificateIdentityPoolsIamV2Api.ListIamV2CertificateIdentityPools(c.caApiContext(ctx), certificateProviderId).PageSize(listIdentityPoolsPageSize).PageToken(pageToken).Execute()
	} else {
		return c.caClient.CertificateIdentityPoolsIamV2Api.ListIamV2CertificateIdentityPools(c.caApiContext(ctx), certificateProviderId).PageSize(listIdentityPoolsPageSize).Execute()
	}
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
