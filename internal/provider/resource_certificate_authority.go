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
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	certificateauthorityv2 "github.com/confluentinc/ccloud-sdk-go-v2/certificate-authority/v2"
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
		CustomizeDiff: certificateAuthorityCustomizeDiff,
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
				Description:  "A PEM encoded string containing the signing certificate chain.",
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
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
					return oldValue == "Local file uploaded" && newValue == ""
				},
				Description: "The url from which to fetch the CRL for the certificate authority. When `crl_chain` is uploaded, the backend reports this as `Local file uploaded`.",
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
				Description:  "A PEM encoded string containing the CRL for this certificate authority.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			paramRequireCrlOnClientCertificate: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to require CRL validation on client certificates.",
			},
		},
	}
}

func certificateAuthorityCustomizeDiff(_ context.Context, d *schema.ResourceDiff, _ interface{}) error {
	triggersBackendCrlUpdate := d.HasChange(paramRequireCrlOnClientCertificate) ||
		d.HasChange(paramCertificateChain) ||
		d.HasChange(paramCrlChain) ||
		d.HasChange(paramCrlUrl)
	if !triggersBackendCrlUpdate {
		return nil
	}
	if d.Get(paramRequireCrlOnClientCertificate).(bool) {
		if err := d.SetNewComputed(paramCrlSource); err != nil {
			return err
		}
		if err := d.SetNewComputed(paramCrlUrl); err != nil {
			return err
		}
		if err := d.SetNewComputed(paramCrlUpdatedAt); err != nil {
			return err
		}
		return nil
	}
	if err := d.SetNew(paramCrlSource, ""); err != nil {
		return err
	}
	if err := d.SetNew(paramCrlUrl, ""); err != nil {
		return err
	}
	if err := d.SetNew(paramCrlUpdatedAt, ""); err != nil {
		return err
	}
	return nil
}

func certificateAuthorityCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*Client)

	createCertificateAuthorityRequest := certificateauthorityv2.NewIamV2CreateCertRequest()
	createCertificateAuthorityRequest.SetDisplayName(d.Get(paramDisplayName).(string))
	createCertificateAuthorityRequest.SetDescription(d.Get(paramDescription).(string))
	createCertificateAuthorityRequest.SetCertificateChain(d.Get(paramCertificateChain).(string))
	createCertificateAuthorityRequest.SetCertificateChainFilename(d.Get(paramCertificateChainFilename).(string))
	requireCrl := d.Get(paramRequireCrlOnClientCertificate).(bool)
	createCertificateAuthorityRequest.SetRequireCrlOnClientCertificate(requireCrl)
	if requireCrl {
		if crlUrl := d.Get(paramCrlUrl).(string); crlUrl != "" {
			createCertificateAuthorityRequest.SetCrlUrl(crlUrl)
		}
		if crlChain := d.Get(paramCrlChain).(string); crlChain != "" {
			createCertificateAuthorityRequest.SetCrlChain(crlChain)
		}
	}

	createCertificateAuthorityRequestJson, err := json.Marshal(createCertificateAuthorityRequest)
	if err != nil {
		return diag.Errorf("error creating Certificate Authority: error marshaling %#v to json: %s", createCertificateAuthorityRequest, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Creating new Certificate Authority: %s", createCertificateAuthorityRequestJson))

	req := c.certificateAuthorityV2Client.CertificateAuthoritiesIamV2Api.CreateIamV2CertificateAuthority(c.certificateAuthorityV2ApiContext(ctx)).IamV2CreateCertRequest(*createCertificateAuthorityRequest)
	createdCertificateAuthority, resp, err := req.Execute()
	if err != nil {
		return diag.Errorf("error creating Certificate Authority %q: %s", createdCertificateAuthority.GetId(), createDescriptiveError(err, resp))
	}
	d.SetId(createdCertificateAuthority.GetId())

	createdCertificateAuthorityJson, err := json.Marshal(createdCertificateAuthority)
	if err != nil {
		return diag.Errorf("error creating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), createdCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished creating Certificate Authority %q: %s", d.Id(), createdCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	// Wait for the backend to finish provisioning and to settle any async CRL-metadata reconciliation
	if err := waitForCertificateAuthorityToProvision(ctx, c, d.Id()); err != nil {
		return diag.Errorf("error waiting for Certificate Authority %q to provision: %s", d.Id(), createDescriptiveError(err))
	}

	return certificateAuthorityRead(ctx, d, meta)
}

func executecertificateAuthorityRead(ctx context.Context, c *Client, certificateAuthorityId string) (certificateauthorityv2.IamV2CertificateAuthority, *http.Response, error) {
	req := c.certificateAuthorityV2Client.CertificateAuthoritiesIamV2Api.GetIamV2CertificateAuthority(c.certificateAuthorityV2ApiContext(ctx), certificateAuthorityId)
	return req.Execute()
}

func certificateAuthorityRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Reading CertificateAuthority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})

	certificateAuthorityId := d.Id()

	if _, err := readCertificateAuthorityAndSetAttributes(ctx, d, meta, certificateAuthorityId); err != nil {
		return diag.FromErr(fmt.Errorf("error reading Certificate Authority %q: %s", certificateAuthorityId, createDescriptiveError(err)))
	}

	if !d.Get(paramRequireCrlOnClientCertificate).(bool) {
		for _, p := range []string{paramCrlSource, paramCrlUrl, paramCrlUpdatedAt} {
			if err := d.Set(p, ""); err != nil {
				return diag.FromErr(err)
			}
		}
	}

	return nil
}

func readCertificateAuthorityAndSetAttributes(ctx context.Context, d *schema.ResourceData, meta interface{}, certificateAuthorityId string) ([]*schema.ResourceData, error) {
	c := meta.(*Client)

	certificateAuthority, resp, err := executecertificateAuthorityRead(c.certificateAuthorityV2ApiContext(ctx), c, certificateAuthorityId)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Error reading Certificate Authority %q: %s", certificateAuthorityId, createDescriptiveError(err, resp)), map[string]interface{}{certificateAuthorityKey: d.Id()})
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

func setCertificateAuthorityAttributes(d *schema.ResourceData, certificateAuthority certificateauthorityv2.IamV2CertificateAuthority) (*schema.ResourceData, error) {
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
	if err := d.Set(paramSerialNumbers, normalizeSerialNumbers(certificateAuthority.GetSerialNumbers())); err != nil {
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
	if err := d.Set(paramRequireCrlOnClientCertificate, certificateAuthority.GetRequireCrlOnClientCertificate()); err != nil {
		return nil, err
	}

	d.SetId(certificateAuthority.GetId())
	return d, nil
}

func certificateAuthorityDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, fmt.Sprintf("Deleting Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})
	c := meta.(*Client)

	req := c.certificateAuthorityV2Client.CertificateAuthoritiesIamV2Api.DeleteIamV2CertificateAuthority(c.certificateAuthorityV2ApiContext(ctx), d.Id())
	_, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error deleting Certificate Authority %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	tflog.Debug(ctx, fmt.Sprintf("Finished deleting Certificate Authority %q", d.Id()), map[string]interface{}{certificateAuthorityKey: d.Id()})

	return nil
}

func certificateAuthorityUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	if d.HasChangesExcept(paramDisplayName, paramDescription, paramCertificateChain, paramCertificateChainFilename, paramCrlUrl, paramCrlChain, paramRequireCrlOnClientCertificate) {
		return diag.Errorf("error updating CertificateAuthority %q: only %q, %q, %q, %q, %q, %q, %q attributes can be updated for Certificate Authority", d.Id(), paramDisplayName, paramDescription, paramCertificateChain, paramCertificateChainFilename, paramCrlUrl, paramCrlChain, paramRequireCrlOnClientCertificate)
	}

	updateCertificateAuthority := certificateauthorityv2.NewIamV2UpdateCertRequest()

	updateCertificateAuthority.SetDisplayName(d.Get(paramDisplayName).(string))
	updateCertificateAuthority.SetDescription(d.Get(paramDescription).(string))
	updateCertificateAuthority.SetCertificateChain(d.Get(paramCertificateChain).(string))
	updateCertificateAuthority.SetCertificateChainFilename(d.Get(paramCertificateChainFilename).(string))
	requireCrl := d.Get(paramRequireCrlOnClientCertificate).(bool)
	updateCertificateAuthority.SetRequireCrlOnClientCertificate(requireCrl)
	if requireCrl {
		if crlUrl := d.Get(paramCrlUrl).(string); crlUrl != "" {
			updateCertificateAuthority.SetCrlUrl(crlUrl)
		}
		if crlChain := d.Get(paramCrlChain).(string); crlChain != "" {
			updateCertificateAuthority.SetCrlChain(crlChain)
		}
	}

	updateCertificateAuthorityJson, err := json.Marshal(updateCertificateAuthority)
	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), updateCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Updating Certificate Authority %q: %s", d.Id(), updateCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	c := meta.(*Client)
	req := c.certificateAuthorityV2Client.CertificateAuthoritiesIamV2Api.UpdateIamV2CertificateAuthority(c.certificateAuthorityV2ApiContext(ctx), d.Id()).IamV2UpdateCertRequest(*updateCertificateAuthority)
	updatedCertificateAuthority, resp, err := req.Execute()

	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: %s", d.Id(), createDescriptiveError(err, resp))
	}

	UpdatedCertificateAuthorityJson, err := json.Marshal(updatedCertificateAuthority)
	if err != nil {
		return diag.Errorf("error updating Certificate Authority %q: error marshaling %#v to json: %s", d.Id(), updatedCertificateAuthority, createDescriptiveError(err))
	}
	tflog.Debug(ctx, fmt.Sprintf("Finished updating Certificate Authority %q: %s", d.Id(), UpdatedCertificateAuthorityJson), map[string]interface{}{certificateAuthorityKey: d.Id()})

	// Wait for the backend to finish reconciliation before Read.
	if err := waitForCertificateAuthorityToProvision(ctx, c, d.Id()); err != nil {
		return diag.Errorf("error waiting for Certificate Authority %q to settle after update: %s", d.Id(), createDescriptiveError(err))
	}

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

func normalizeSerialNumbers(serials []string) []string {
	out := make([]string, len(serials))
	for i, s := range serials {
		out[i] = normalizeSerialNumber(s)
	}
	return out
}

func normalizeSerialNumber(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	upper := strings.ToUpper(s)
	containsHexLetters := false
	for _, c := range upper {
		if c >= 'A' && c <= 'F' {
			containsHexLetters = true
			break
		}
	}
	if containsHexLetters {
		if i, ok := new(big.Int).SetString(upper, 16); ok {
			return strings.ToUpper(i.Text(16))
		}
		return upper
	}
	if i, ok := new(big.Int).SetString(s, 10); ok {
		return strings.ToUpper(i.Text(16))
	}
	if i, ok := new(big.Int).SetString(s, 16); ok {
		return strings.ToUpper(i.Text(16))
	}
	return upper
}
