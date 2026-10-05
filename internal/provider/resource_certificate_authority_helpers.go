// Copyright 2026 Confluent Inc. All Rights Reserved.
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
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// certificateAuthorityCustomizeDiff reconciles the CRL-derived computed fields
// (crl_source, crl_updated_at, and crl_url when the backend supplies it) whenever a
// change would trigger a backend CRL update. When CRL validation is turned off, it
// clears crl_source and crl_url; crl_updated_at is deliberately left as-is (the
// backend's last-known value), not reset. Referenced by name via terraform.customize_diff
// in cli-terraform-generator's registry.yaml -- kept hand-written because it is
// cross-field diff logic the spec cannot express.
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
		if err := d.SetNewComputed(paramCrlUpdatedAt); err != nil {
			return err
		}
		if d.Get(paramCrlUrl).(string) == "" {
			if err := d.SetNewComputed(paramCrlUrl); err != nil {
				return err
			}
		}
		return nil
	}
	for _, p := range []string{paramCrlSource, paramCrlUrl} {
		if err := d.SetNew(p, ""); err != nil {
			return err
		}
	}
	return nil
}

// suppressCrlUrlLocalFilePlaceholder suppresses the diff between the backend's
// "Local file uploaded" sentinel (crlUrlLocalFilePlaceholder, constants.go) and an
// unset config value, so uploading crl_chain doesn't show perpetual drift on
// crl_url. Explicit URLs (newValue != "") still diff normally. Referenced by name via
// terraform.diff_suppress_overrides.crl_url; extracted from what was previously an
// inline anonymous closure on the hand-written schema.
func suppressCrlUrlLocalFilePlaceholder(_, oldValue, newValue string, _ *schema.ResourceData) bool {
	return oldValue == crlUrlLocalFilePlaceholder && newValue == ""
}

// convertTimeToStringSlice renders a []time.Time SDK field (expiration_dates,
// serial_numbers) as the []string Terraform's TypeSet expects. Referenced by
// generated read-back code for any array-of-date-time attribute
// (AttributeMetadata.IsSDKTimeArray / TFSetValueExpr in cli-terraform-generator).
func convertTimeToStringSlice(timeValues []time.Time) []string {
	s := make([]string, len(timeValues))
	for i, timeValue := range timeValues {
		s[i] = fmt.Sprint(timeValue)
	}
	return s
}
