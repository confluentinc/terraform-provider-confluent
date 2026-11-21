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
	"reflect"
	"strings"
	"testing"

	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// The cty types below have the same shape as the raw config of features.
var (
	testShreddingType = cty.Object(map[string]cty.Type{paramEnabled: cty.Bool})
	testFeatureType   = cty.Object(map[string]cty.Type{
		paramFeature:   cty.String,
		paramEnabled:   cty.Bool,
		paramShredding: cty.List(testShreddingType),
	})
)

// testFeature returns a features element. A shredding argument adds a shredding block with that enabled value.
func testFeature(name string, enabled bool, shredding ...cty.Value) cty.Value {
	shreddingList := cty.ListValEmpty(testShreddingType)
	if len(shredding) > 0 {
		shreddingList = cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{paramEnabled: shredding[0]})})
	}
	return cty.ObjectVal(map[string]cty.Value{
		paramFeature:   cty.StringVal(name),
		paramEnabled:   cty.BoolVal(enabled),
		paramShredding: shreddingList,
	})
}

func testFeatures(elements ...cty.Value) cty.Value {
	if len(elements) == 0 {
		return cty.ListValEmpty(testFeatureType)
	}
	return cty.ListVal(elements)
}

func TestExpandFeaturesSendsOnlyConfiguredEntries(t *testing.T) {
	t.Run("VARIANT without shredding", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{paramFeatures: testFeatures(testFeature(variantFeature, true))})
		got := expandFeatures(raw)
		if len(got) != 1 || got[0].TableflowV1VariantFeature == nil {
			t.Fatalf("got %+v, want 1 VARIANT entry", got)
		}
		variant := got[0].TableflowV1VariantFeature
		if variant.GetFeature() != "VARIANT" || !variant.GetEnabled() {
			t.Errorf("got %+v, want VARIANT, enabled true", variant)
		}
		if variant.Shredding != nil {
			t.Errorf("got shredding %+v, want nil", variant.Shredding)
		}
	})
	t.Run("VARIANT with shredding", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramFeatures: testFeatures(testFeature(variantFeature, true, cty.True)),
		})
		got := expandFeatures(raw)
		if len(got) != 1 || got[0].TableflowV1VariantFeature == nil {
			t.Fatalf("got %+v, want 1 VARIANT entry", got)
		}
		shredding := got[0].TableflowV1VariantFeature.Shredding
		if shredding == nil || shredding.Enabled == nil || !*shredding.Enabled {
			t.Errorf("got shredding %+v, want enabled true", shredding)
		}
	})
	t.Run("VARIANT with a shredding block that has no enabled value", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramFeatures: testFeatures(testFeature(variantFeature, true, cty.NullVal(cty.Bool))),
		})
		got := expandFeatures(raw)
		if len(got) != 1 || got[0].TableflowV1VariantFeature == nil {
			t.Fatalf("got %+v, want 1 VARIANT entry", got)
		}
		if got[0].TableflowV1VariantFeature.Shredding != nil {
			t.Errorf("got shredding %+v, want nil", got[0].TableflowV1VariantFeature.Shredding)
		}
	})
	t.Run("no features block", func(t *testing.T) {
		for name, raw := range map[string]cty.Value{
			"attribute not in the object": cty.EmptyObjectVal,
			"empty list":                  cty.ObjectVal(map[string]cty.Value{paramFeatures: testFeatures()}),
			"null list":                   cty.ObjectVal(map[string]cty.Value{paramFeatures: cty.NullVal(cty.List(testFeatureType))}),
		} {
			if got := expandFeatures(raw); got != nil {
				t.Errorf("%s: got %+v, want nil", name, got)
			}
		}
	})
}

func TestFlattenFeatures(t *testing.T) {
	variant := tableflowv1.NewTableflowV1VariantFeature("VARIANT", true)
	variant.SetShredding(tableflowv1.TableflowV1VariantShredding{Enabled: tableflowv1.PtrBool(false)})
	features := []tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf{
		tableflowv1.TableflowV1VariantFeatureAsTableflowV1TableFlowTopicConfigsSpecFeaturesOneOf(variant),
	}

	t.Run("VARIANT with shredding", func(t *testing.T) {
		want := []interface{}{map[string]interface{}{
			paramFeature:   "VARIANT",
			paramEnabled:   true,
			paramShredding: []interface{}{map[string]interface{}{paramEnabled: false}},
		}}
		if got := flattenFeatures(features, nil); !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
		if got := flattenFeatures(features, map[string]bool{"VARIANT": true}); !reflect.DeepEqual(got, want) {
			t.Errorf("keep VARIANT: got %#v, want %#v", got, want)
		}
	})
	t.Run("keep filters out VARIANT", func(t *testing.T) {
		for name, keep := range map[string]map[string]bool{
			"empty keep":       {},
			"keep other names": {"OTHER": true},
		} {
			if got := flattenFeatures(features, keep); len(got) != 0 {
				t.Errorf("%s: got %#v, want no VARIANT element", name, got)
			}
		}
	})
	t.Run("flatten skips an entry that the SDK does not know", func(t *testing.T) {
		unknown := []tableflowv1.TableflowV1TableFlowTopicConfigsSpecFeaturesOneOf{{}}
		if got := flattenFeatures(unknown, nil); len(got) != 0 {
			t.Errorf("got %#v, want no element", got)
		}
	})
}

func TestFeatureNamesInState(t *testing.T) {
	s := map[string]*schema.Schema{paramFeatures: featuresSchema()}

	if got := featureNamesInState(schema.TestResourceDataRaw(t, s, map[string]interface{}{})); got != nil {
		t.Errorf("no features: got %#v, want nil", got)
	}

	d := schema.TestResourceDataRaw(t, s, map[string]interface{}{
		paramFeatures: []interface{}{map[string]interface{}{paramFeature: "VARIANT", paramEnabled: true}},
	})
	if got, want := featureNamesInState(d), map[string]bool{"VARIANT": true}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

// TestFeaturesDiff plans a features config against a features state on the SDK plan path. A change of enabled
// or shredding must make a diff, and the computed shredding value or a removed block must not.
func TestFeaturesDiff(t *testing.T) {
	r := &schema.Resource{Schema: map[string]*schema.Schema{paramFeatures: featuresSchema()}}
	variantState := func(enabled, shredding bool) map[string]interface{} {
		return map[string]interface{}{paramFeatures: []interface{}{map[string]interface{}{
			paramFeature:   variantFeature,
			paramEnabled:   enabled,
			paramShredding: []interface{}{map[string]interface{}{paramEnabled: shredding}},
		}}}
	}
	tests := []struct {
		name     string
		state    map[string]interface{}
		features cty.Value
		want     map[string]string
	}{
		{"shredding not configured, state false", variantState(true, false), testFeatures(testFeature(variantFeature, true)), nil},
		{"shredding not configured, state true", variantState(true, true), testFeatures(testFeature(variantFeature, true)), nil},
		{"features block removed", variantState(true, true), testFeatures(), nil},
		{"same values", variantState(true, true), testFeatures(testFeature(variantFeature, true, cty.True)), nil},
		{"shredding false to true", variantState(true, false), testFeatures(testFeature(variantFeature, true, cty.True)),
			map[string]string{"features.0.shredding.0.enabled": "true"}},
		{"enabled false to true", variantState(false, false), testFeatures(testFeature(variantFeature, true)),
			map[string]string{"features.0.enabled": "true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, r.Schema, tt.state)
			d.SetId("tt-abc123")
			raw := cty.ObjectVal(map[string]cty.Value{paramFeatures: tt.features})
			state := d.State()
			state.RawConfig = raw
			diff, err := r.SimpleDiff(context.Background(), state, terraform.NewResourceConfigShimmed(raw, r.CoreConfigSchema()), nil)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]string{}
			if diff != nil {
				for key, attr := range diff.Attributes {
					if strings.HasPrefix(key, paramFeatures+".") {
						got[key] = attr.New
					}
				}
			}
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got diff %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFeaturesPlanChecks(t *testing.T) {
	checkTableflowTopicError(t, "checkShredding(false, ptr(true))", checkShredding(false, testBoolPtr(true)),
		"shredding can't be enabled while VARIANT is disabled.")
	checkTableflowTopicError(t, "checkShredding(false, ptr(false))", checkShredding(false, testBoolPtr(false)), "")
	checkTableflowTopicError(t, "checkShredding(false, nil)", checkShredding(false, nil), "")
	checkTableflowTopicError(t, "checkShredding(true, ptr(true))", checkShredding(true, testBoolPtr(true)), "")
}

func TestTableflowTopicFeaturesCustomizeDiff(t *testing.T) {
	r := tableflowTopicTestResource()
	r.CustomizeDiff = tableflowTopicFeaturesCustomizeDiff

	featuresState := map[string]interface{}{
		paramFeatures: []interface{}{map[string]interface{}{
			paramFeature:   "VARIANT",
			paramEnabled:   true,
			paramShredding: []interface{}{map[string]interface{}{paramEnabled: true}},
		}},
	}

	tests := []struct {
		name   string
		state  map[string]interface{}
		config map[string]cty.Value
		want   string
	}{
		{
			name:  "update: a removed features block gives no error",
			state: featuresState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(3)), testNoDelta),
			},
		},
		{
			name:   "update: a disabled VARIANT gives no plan error, the API checks it",
			state:  featuresState,
			config: map[string]cty.Value{paramFeatures: testFeatures(testFeature(variantFeature, false))},
		},
		{
			name:   "create: shredding on a disabled VARIANT",
			config: map[string]cty.Value{paramFeatures: testFeatures(testFeature(variantFeature, false, cty.True))},
			want:   "shredding can't be enabled while VARIANT is disabled.",
		},
		{
			name:   "update: shredding on a disabled VARIANT",
			state:  featuresState,
			config: map[string]cty.Value{paramFeatures: testFeatures(testFeature(variantFeature, false, cty.True))},
			want:   "shredding can't be enabled while VARIANT is disabled.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := planTableflowTopicTestResource(t, r, tt.state, tt.config)
			switch {
			case tt.want == "" && err != nil:
				t.Errorf("got %q, want nil", err)
			case tt.want != "" && err == nil:
				t.Errorf("got nil, want %q", tt.want)
			case tt.want != "" && err.Error() != tt.want:
				t.Errorf("got %q, want %q", err, tt.want)
			}
		})
	}
}
