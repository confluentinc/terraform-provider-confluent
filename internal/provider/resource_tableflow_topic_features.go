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

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

func expandFeatures(raw cty.Value) []tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf {
	features := ctyAttr(raw, paramFeatures)
	if features.IsNull() || !features.IsKnown() || !features.Type().IsListType() {
		return nil
	}
	var result []tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf
	for it := features.ElementIterator(); it.Next(); {
		_, element := it.Element()
		name, nameOk := ctyString(ctyAttr(element, paramFeature))
		enabled, enabledOk := ctyBool(ctyAttr(element, paramEnabled))
		if !nameOk || !enabledOk {
			continue
		}
		// Each feature has its own oneOf type in the SDK. The SDK has a type only for VARIANT now.
		if name == variantFeature {
			result = append(result, variantEntry(element, enabled))
		}
	}
	return result
}

// variantEntry returns the VARIANT entry of features for one features block. The entry has shredding only when the
// block sets shredding.enabled.
func variantEntry(element cty.Value, enabled bool) tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf {
	variant := tableflowv1.NewTableflowV1VariantFeature(variantFeature, enabled)
	if shredding, ok := ctyBool(ctyAttr(ctyFirstBlock(ctyAttr(element, paramShredding)), paramEnabled)); ok {
		variant.SetShredding(tableflowv1.TableflowV1VariantShredding{Enabled: &shredding})
	}
	return tableflowv1.TableflowV1VariantFeatureAsTableflowV1TableFlowTopicConfigsSpecFeaturesOneOf(variant)
}

func flattenFeatures(features []tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf, keep map[string]bool) []interface{} {
	var result []interface{}
	for _, entry := range features {
		variant := entry.TableflowV1VariantFeature
		if variant == nil || (keep != nil && !keep[variantFeature]) {
			continue
		}
		shredding := []interface{}{}
		if value, ok := variant.GetShreddingOk(); ok {
			shredding = []interface{}{map[string]interface{}{paramEnabled: value.GetEnabled()}}
		}
		result = append(result, map[string]interface{}{
			paramFeature:   variantFeature,
			paramEnabled:   variant.GetEnabled(),
			paramShredding: shredding,
		})
	}
	return result
}

// featureNamesInState returns the feature names in the features attribute of d. It returns nil when d has no features.
func featureNamesInState(d *schema.ResourceData) map[string]bool {
	features, ok := d.Get(paramFeatures).([]interface{})
	if !ok {
		return nil
	}
	var names map[string]bool
	for _, item := range features {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := m[paramFeature].(string); ok && name != "" {
			if names == nil {
				names = map[string]bool{}
			}
			names[name] = true
		}
	}
	return names
}

// checkShredding checks that shredding is not enabled on a disabled VARIANT. nil means that there is no value. The API
// accepts a false value on a disabled VARIANT, so that a client can send back the result of a GET.
func checkShredding(featureEnabled bool, shredding *bool) error {
	if shredding != nil && *shredding && !featureEnabled {
		return fmt.Errorf("shredding can't be enabled while %s is disabled.", variantFeature)
	}
	return nil
}

// tableflowTopicFeaturesCustomizeDiff does the plan-time check of features: shredding can't be enabled while VARIANT
// is disabled.
func tableflowTopicFeaturesCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	raw := diff.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() {
		return nil
	}
	if !ctyAttr(raw, paramFeatures).IsWhollyKnown() {
		tflog.Debug(ctx, fmt.Sprintf("Skipping the plan checks of Tableflow Topic %q: %q is not known yet", diff.Id(), paramFeatures))
		return nil
	}

	for _, entry := range expandFeatures(raw) {
		if variant := entry.TableflowV1VariantFeature; variant != nil {
			if err := checkShredding(variant.GetEnabled(), variant.GetShredding().Enabled); err != nil {
				return err
			}
		}
	}
	return nil
}

// ctyString returns the value of a known string, and false for ok when v is null, unknown or not a string.
func ctyString(v cty.Value) (string, bool) {
	if v.IsNull() || !v.IsKnown() || !v.Type().Equals(cty.String) {
		return "", false
	}
	return v.AsString(), true
}
