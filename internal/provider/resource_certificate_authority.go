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
	"time"

	ca "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	paramCertificateChain         = "certificate_chain"
	paramCertificateChainFilename = "certificate_chain_filename"
	paramCrlSource                = "crl_source"
	paramCrlUrl                   = "crl_url"
	paramCrlChain                 = "crl_chain"
	paramCrlUpdatedAt             = "crl_updated_at"
	paramFingerprints             = "fingerprints"
	paramExpirationDates          = "expiration_dates"
	paramSerialNumbers            = "serial_numbers"
)

func certificateAuthorityResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: certificateAuthorityCreate,
		ReadContext:   certificateAuthorityRead,
		UpdateContext: certificateAuthorityUpdate,
		DeleteContext: certificateAuthorityDelete,
		Importer: &schema.ResourceImporter{
			StateContext: certificateAuthorityImport,
		},
		Schema: map[string]*schema.Schema{
			paramDisplayName: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A name for the Certificate Authority.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramDescription: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "A description of the Certificate Authority.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramCertificateChain: {
				Type:         schema.TypeString,
				Required:     true,
				Sensitive:    true,
				Description:  "A base64 encoded string containing the signing certificate chain.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramCertificateChainFilename: {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The name of the certificate file.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramFingerprints: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The fingerprints for each certificate in the certificate chain.",
			},
			paramExpirationDates: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The expiration dates of certificates in the chain.",
			},
			paramSerialNumbers: {
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Computed:    true,
				Description: "The serial numbers for each certificate in the certificate chain.",
			},
			paramCrlSource: {
				Type:     schema.TypeString,
				Computed: true,
			},
			paramCrlUrl: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The url from which to fetch the CRL for the certificate authority.",
			},
			paramCrlUpdatedAt: {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The timestamp for when CRL was last updated.",
			},
			paramCrlChain: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				Description:  "A base64 encoded string containing the CRL for this certificate authority.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
		},
	}
}

func certificateAuthorityCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	createCertificateAuthorityRequest := ca.NewIamV2CreateCertRequest()
	createCertificateAuthorityRequest.SetDisplayName(d.Get(paramDisplayName).(string))
	createCertificateAuthorityRequest.SetDescription(d.Get(paramDescription).(string))
	createCertificateAuthorityRequest.SetCertificateChain(d.Get(paramCertificateChain).(string))
	createCertificateAuthorityRequest.SetCertificateChainFilename(d.Get(paramCertificateChainFilename).(string))
	createCertificateAuthorityRequest.SetCrlUrl(d.Get(paramCrlUrl).(string))
	createCertificateAuthorityRequest.SetCrlChain(d.Get(paramCrlChain).(string))

	createCertificateAuthorityRequestJson, err := json.Marshal(createCertificateAuthorityRequest)
	if err != nil {
		return diag.Errorf("error creating Certificate Authority: error marshaling %#v to json: %s", createCertificateAuthorityRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Certificate Authority: %s", createCertificateAuthorityRequestJson))

	req := c.caClient.CertificateAuthoritiesIamV2Api.CreateIamV2CertificateAuthority(c.caApiContext(ctx)).IamV2CreateCertRequest(*createCertificateAuthorityRequest)
	createdCertificateAuthority, _, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Certificate Authority %q: %s", createdCertificateAuthority.GetId(), createDescriptiveError(err))
	}
	d.SetId(createdCertificateAuthority.GetId())

	createdCertificateAuthorityJson, err := json.Marshal(createdCertificateAuthority)
	if err != nil {
		return diag.Errorf("error creating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), createdCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Certificate Authority %q: %s", d.Id(), createdCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	return certificateAuthorityRead(ctx, d, meta)
}

func executecertificateAuthorityRead(ctx context.Context, c *Client, certificateAuthorityId string) (ca.IamV2CertificateAuthority, *http.Response, error) {
	req := c.caClient.CertificateAuthoritiesIamV2Api.GetIamV2CertificateAuthority(c.caApiContext(ctx), certificateAuthorityId)
	return req.Execute()
}

func certificateAuthorityRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading CertificateAuthority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})

	certificateAuthorityId := d.Id()

	if _, err := readCertificateAuthorityAndSetAttributes(ctx, d, meta, certificateAuthorityId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Certificate Authority %q: %s", certificateAuthorityId, createDescriptiveError(err)))
	}

	return nil
}

func readCertificateAuthorityAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, certificateAuthorityId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	certificateAuthority, resp, err := executecertificateAuthorityRead(c.caApiContext(ctx), c, certificateAuthorityId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Certificate Authority %q: %s", certificateAuthorityId, createDescriptiveError(err)), map[string]interface{}{certificateAuthorityKey: d.Id()})
		isResourceNotFound := isNonKafkaRestApiResourceNotFound(resp)
		if isResourceNotFound && !d.IsNewResource() {
			tflog.Warn(ctx, fmt.Sprintf("Removing Certificate Authority %q in TF state because Certificate Authority could not be found on the server", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})
			d.SetId("")
			return nil, nil
		}

		return nil, err
	}
	certificateAuthorityJson, err := json.Marshal(certificateAuthority)
	if err != nil {
		return nil, fmt.Errorf("error reading Certificate Authority %q: error marshaling %#v to json: %s", certificateAuthorityId, certificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Fetched Certificate Authority %q: %s", d.Id(), certificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	if _, err := setCertificateAuthorityAttributes(d, certificateAuthority); err != nil {
		return nil, createDescriptiveError(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished reading Certificate Authority %q", certificateAuthorityId), map[string]interface{}{certificateAuthorityKey: d.Id()})

	return []*schema.ResourceData{d}, nil
}

func setCertificateAuthorityAttributes(d *schema.ResourceData, certificateAuthority ca.IamV2CertificateAuthority) (*schema.ResourceData, error) {
	if err := d.Set(paramDisplayName, certificateAuthority.GetDisplayName()); err != nil {
		return nil, err
	}
	if err := d.Set(paramDescription, certificateAuthority.GetDescription()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCertificateChainFilename, certificateAuthority.GetCertificateChainFilename()); err != nil {
		return nil, err
	}
	if err := d.Set(paramFingerprints, certificateAuthority.GetFingerprints()); err != nil {
		return nil, err
	}
	if err := d.Set(paramExpirationDates, convertTimeToStringSlice(certificateAuthority.GetExpirationDates())); err != nil {
		return nil, err
	}
	if err := d.Set(paramSerialNumbers, certificateAuthority.GetSerialNumbers()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCrlSource, certificateAuthority.GetCrlSource()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCrlUrl, certificateAuthority.GetCrlUrl()); err != nil {
		return nil, err
	}
	if err := d.Set(paramCrlUpdatedAt, fmt.Sprint(certificateAuthority.GetCrlUpdatedAt())); err != nil {
		return nil, err
	}

	d.SetId(certificateAuthority.GetId())
	return d, nil
}

func certificateAuthorityDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})
	c := meta.(*Client)

	req := c.caClient.CertificateAuthoritiesIamV2Api.DeleteIamV2CertificateAuthority(c.caApiContext(ctx), d.Id())
	_, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Certificate Authority %q: %s", d.Id(), createDescriptiveError(err))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})

	return nil
}

func certificateAuthorityUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramCertificateChain, paramCertificateChainFilename, paramCrlUrl, paramCrlChain) {
		return diag.Errorf("error updating CertificateAuthority %q: only %q, %q, %q, %q, %q, %q attributes can be updated for Certificate Authority", d.Id(), paramDisplayName, paramDescription, paramCertificateChain, paramCertificateChainFilename, paramCrlUrl, paramCrlChain)
	}

	updateCertificateAuthority := ca.NewIamV2UpdateCertRequest()

	updateCertificateAuthority.SetDisplayName(d.Get(paramDisplayName).(string))
	updateCertificateAuthority.SetDescription(d.Get(paramDescription).(string))
	updateCertificateAuthority.SetCertificateChain(d.Get(paramCertificateChain).(string))
	updateCertificateAuthority.SetCertificateChainFilename(d.Get(paramCertificateChainFilename).(string))
	updateCertificateAuthority.SetCrlUrl(d.Get(paramCrlUrl).(string))
	updateCertificateAuthority.SetCrlChain(d.Get(paramCrlChain).(string))

	updateCertificateAuthorityJson, err := json.Marshal(updateCertificateAuthority)
	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), updateCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Certificate Authority %q: %s", d.Id(), updateCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	c := meta.(*Client)
	req := c.caClient.CertificateAuthoritiesIamV2Api.UpdateIamV2CertificateAuthority(c.caApiContext(ctx), d.Id()).IamV2UpdateCertRequest(*updateCertificateAuthority)
	updatedCertificateAuthority, _, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: %s", d.Id(), createDescriptiveError(err))
	}

	UpdatedCertificateAuthorityJson, err := json.Marshal(updatedCertificateAuthority)
	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), updatedCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Certificate Authority %q: %s", d.Id(), UpdatedCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})
	return certificateAuthorityRead(ctx, d, meta)
}

func certificateAuthorityImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Debug(ctx, fmt.Sprintf("Importing Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})

	// Mark resource as new to avoid d.Set("") when getting 404
	d.MarkNewResource()
	if _, err := readCertificateAuthorityAndSetAttributes(ctx, d, meta, d.Id()); err != nil {
		return nil, fmt.Errorf("error importing Certificate Authority %q: %s", d.Id(), err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished importing Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})
	return []*schema.ResourceData{d}, nil
}

func convertTimeToStringSlice(timeValues []time.Time) []string {
	s := make([]string, len(timeValues))
	for i, timeValue := range timeValues {
		s[i] = fmt.Sprint(timeValue)
	}
	return s
}
