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
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
	"github.com/hashicorp/go-cty/cty"
)

// The cty types below have the same shape as the raw config of table_format_properties.
var (
	testIcebergType = cty.Object(map[string]cty.Type{paramEnabled: cty.Bool, paramFormatVersion: cty.Number})
	testDeltaType   = cty.Object(map[string]cty.Type{paramEnabled: cty.Bool})
	testNoIceberg   = cty.ListValEmpty(testIcebergType)
	testNoDelta     = cty.ListValEmpty(testDeltaType)
)

func testIcebergList(enabled bool, formatVersion cty.Value) cty.Value {
	return cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
		paramEnabled:       cty.BoolVal(enabled),
		paramFormatVersion: formatVersion,
	})})
}

func testDeltaList(enabled bool) cty.Value {
	return cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{paramEnabled: cty.BoolVal(enabled)})})
}

func testTableFormatProperties(icebergList, deltaList cty.Value) cty.Value {
	return cty.ListVal([]cty.Value{cty.ObjectVal(map[string]cty.Value{
		paramIceberg: icebergList,
		paramDelta:   deltaList,
	})})
}

func testTableFormats(formats ...string) cty.Value {
	values := make([]cty.Value, len(formats))
	for i, format := range formats {
		values[i] = cty.StringVal(format)
	}
	return cty.SetVal(values)
}

func TestExpandTableFormatPropertiesForCreate(t *testing.T) {
	t.Run("ICEBERG with format_version", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(3)), testNoDelta),
		})
		got := expandTableFormatPropertiesForCreate(raw)
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		iceberg := got[0].TableflowV1IcebergTableFormatProperties
		if iceberg == nil || got[0].TableflowV1DeltaTableFormatProperties != nil {
			t.Fatalf("got %+v, want only an ICEBERG entry", got[0])
		}
		if iceberg.GetFormat() != "ICEBERG" || !iceberg.GetEnabled() || iceberg.FormatVersion == nil || *iceberg.FormatVersion != 3 {
			t.Errorf("got %+v, want ICEBERG, enabled true, format_version 3", iceberg)
		}
	})
	t.Run("ICEBERG without format_version", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NullVal(cty.Number)), testNoDelta),
		})
		got := expandTableFormatPropertiesForCreate(raw)
		if len(got) != 1 || got[0].TableflowV1IcebergTableFormatProperties == nil {
			t.Fatalf("got %+v, want 1 ICEBERG entry", got)
		}
		if got[0].TableflowV1IcebergTableFormatProperties.FormatVersion != nil {
			t.Errorf("got format_version %d, want nil", *got[0].TableflowV1IcebergTableFormatProperties.FormatVersion)
		}
	})
	t.Run("DELTA disabled", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testNoIceberg, testDeltaList(false)),
		})
		got := expandTableFormatPropertiesForCreate(raw)
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		delta := got[0].TableflowV1DeltaTableFormatProperties
		if delta == nil || got[0].TableflowV1IcebergTableFormatProperties != nil {
			t.Fatalf("got %+v, want only a DELTA entry", got[0])
		}
		if delta.GetFormat() != "DELTA" || delta.GetEnabled() {
			t.Errorf("got %+v, want DELTA, enabled false", delta)
		}
		body, err := json.Marshal(got[0])
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		if !strings.Contains(string(body), `"enabled":false`) {
			t.Errorf("got JSON %s, want it to have \"enabled\":false", body)
		}
	})
	t.Run("no table_format_properties block", func(t *testing.T) {
		for name, raw := range map[string]cty.Value{
			"attribute not in the object": cty.EmptyObjectVal,
			"null config":                 cty.NullVal(cty.EmptyObject),
			"empty block list": cty.ObjectVal(map[string]cty.Value{
				paramTableFormatProperties: cty.ListValEmpty(testTableFormatProperties(testNoIceberg, testNoDelta).Type().ElementType()),
			}),
		} {
			if got := expandTableFormatPropertiesForCreate(raw); got != nil {
				t.Errorf("%s: got %+v, want nil", name, got)
			}
		}
	})
	t.Run("table_format_properties without a sub-block", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testNoIceberg, testNoDelta),
		})
		if got := expandTableFormatPropertiesForCreate(raw); got != nil {
			t.Errorf("got %+v, want nil (never an empty slice)", got)
		}
	})
}

func TestExpandTableFormatPropertiesForUpdate(t *testing.T) {
	t.Run("ICEBERG with format_version", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(3)), testNoDelta),
		})
		got := expandTableFormatPropertiesForUpdate(raw)
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		iceberg := got[0].TableflowV1IcebergTableFormatProperties
		if iceberg == nil || got[0].TableflowV1DeltaTableFormatProperties != nil {
			t.Fatalf("got %+v, want only an ICEBERG entry", got[0])
		}
		if iceberg.GetFormat() != "ICEBERG" || !iceberg.GetEnabled() || iceberg.FormatVersion == nil || *iceberg.FormatVersion != 3 {
			t.Errorf("got %+v, want ICEBERG, enabled true, format_version 3", iceberg)
		}
	})
	t.Run("ICEBERG without format_version", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NullVal(cty.Number)), testNoDelta),
		})
		got := expandTableFormatPropertiesForUpdate(raw)
		if len(got) != 1 || got[0].TableflowV1IcebergTableFormatProperties == nil {
			t.Fatalf("got %+v, want 1 ICEBERG entry", got)
		}
		if got[0].TableflowV1IcebergTableFormatProperties.FormatVersion != nil {
			t.Errorf("got format_version %d, want nil", *got[0].TableflowV1IcebergTableFormatProperties.FormatVersion)
		}
	})
	t.Run("DELTA disabled", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testNoIceberg, testDeltaList(false)),
		})
		got := expandTableFormatPropertiesForUpdate(raw)
		if len(got) != 1 {
			t.Fatalf("got %d entries, want 1", len(got))
		}
		delta := got[0].TableflowV1DeltaTableFormatProperties
		if delta == nil || got[0].TableflowV1IcebergTableFormatProperties != nil {
			t.Fatalf("got %+v, want only a DELTA entry", got[0])
		}
		if delta.GetFormat() != "DELTA" || delta.GetEnabled() {
			t.Errorf("got %+v, want DELTA, enabled false", delta)
		}
		body, err := json.Marshal(got[0])
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		if !strings.Contains(string(body), `"enabled":false`) {
			t.Errorf("got JSON %s, want it to have \"enabled\":false", body)
		}
	})
	t.Run("both sub-blocks", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testIcebergList(false, cty.NullVal(cty.Number)), testDeltaList(true)),
		})
		if got := expandTableFormatPropertiesForUpdate(raw); len(got) != 2 {
			t.Errorf("got %d entries, want 2", len(got))
		}
	})
	t.Run("no table_format_properties block", func(t *testing.T) {
		if got := expandTableFormatPropertiesForUpdate(cty.EmptyObjectVal); got != nil {
			t.Errorf("got %+v, want nil", got)
		}
	})
	t.Run("table_format_properties without a sub-block", func(t *testing.T) {
		raw := cty.ObjectVal(map[string]cty.Value{
			paramTableFormatProperties: testTableFormatProperties(testNoIceberg, testNoDelta),
		})
		if got := expandTableFormatPropertiesForUpdate(raw); got != nil {
			t.Errorf("got %+v, want nil (never an empty slice)", got)
		}
	})
}

func TestFlattenTableFormatProperties(t *testing.T) {
	t.Run("DELTA and ICEBERG", func(t *testing.T) {
		props := []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf{
			tableflowv1.TableflowV1DeltaTableFormatPropertiesAsTableflowV1TableflowTopicSpecTableFormatPropertiesOneOf(
				tableflowv1.NewTableflowV1DeltaTableFormatProperties("DELTA", false)),
			tableflowv1.TableflowV1IcebergTableFormatPropertiesAsTableflowV1TableflowTopicSpecTableFormatPropertiesOneOf(
				&tableflowv1.TableflowV1IcebergTableFormatProperties{Format: "ICEBERG", Enabled: true, FormatVersion: tableflowv1.PtrInt32(3)}),
		}
		want := []interface{}{map[string]interface{}{
			paramIceberg: []interface{}{map[string]interface{}{paramEnabled: true, paramFormatVersion: 3}},
			paramDelta:   []interface{}{map[string]interface{}{paramEnabled: false}},
		}}
		if got := flattenTableFormatProperties(props); !reflect.DeepEqual(got, want) {
			t.Errorf("got %#v, want %#v", got, want)
		}
	})
	t.Run("ICEBERG disabled without format_version", func(t *testing.T) {
		props := []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf{
			tableflowv1.TableflowV1IcebergTableFormatPropertiesAsTableflowV1TableflowTopicSpecTableFormatPropertiesOneOf(
				tableflowv1.NewTableflowV1IcebergTableFormatProperties("ICEBERG", false)),
		}
		got := flattenTableFormatProperties(props)
		if len(got) != 1 {
			t.Fatalf("got %#v, want 1 element", got)
		}
		iceberg := got[0].(map[string]interface{})[paramIceberg].([]interface{})[0].(map[string]interface{})
		if iceberg[paramEnabled] != false || iceberg[paramFormatVersion] != 0 {
			t.Errorf("got %#v, want enabled false, format_version 0", iceberg)
		}
	})
	t.Run("flatten skips an entry that the SDK does not know", func(t *testing.T) {
		props := []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf{{}}
		if got := flattenTableFormatProperties(props); len(got) != 0 {
			t.Errorf("got %#v, want no element", got)
		}
	})
}

func TestTableFormatPropertiesPlanChecks(t *testing.T) {
	checkTableflowTopicError(t, "checkFormatVersionNotLowered(3, 2)", checkFormatVersionNotLowered(3, 2),
		"format_version can't be lowered from 3 to 2.")
	checkTableflowTopicError(t, "checkFormatVersionNotLowered(0, 2)", checkFormatVersionNotLowered(0, 2), "")
	checkTableflowTopicError(t, "checkFormatVersionNotLowered(2, 3)", checkFormatVersionNotLowered(2, 3), "")

	checkTableflowTopicError(t, "checkDisabledIcebergVersion(false, 3, 3)", checkDisabledIcebergVersion(false, 3, 3), "")
	checkTableflowTopicError(t, "checkDisabledIcebergVersion(false, 3, 2)", checkDisabledIcebergVersion(false, 3, 2),
		"format_version can't change while ICEBERG is disabled.")
	checkTableflowTopicError(t, "checkDisabledIcebergVersion(false, 0, 2)", checkDisabledIcebergVersion(false, 0, 2),
		"format_version can't change while ICEBERG is disabled.")
	checkTableflowTopicError(t, "checkDisabledIcebergVersion(true, 2, 3)", checkDisabledIcebergVersion(true, 2, 3), "")
}

func TestTableflowTopicTableFormatPropertiesCustomizeDiff(t *testing.T) {
	r := tableflowTopicTestResource()
	r.CustomizeDiff = tableflowTopicTableFormatPropertiesCustomizeDiff

	icebergState := map[string]interface{}{
		paramTableFormats: []interface{}{"ICEBERG"},
		paramTableFormatProperties: []interface{}{map[string]interface{}{
			paramIceberg: []interface{}{map[string]interface{}{paramEnabled: true, paramFormatVersion: 3}},
			paramDelta:   []interface{}{map[string]interface{}{paramEnabled: false}},
		}},
	}
	// A DELTA topic with ICEBERG disabled at a stored version 2.
	deltaState := map[string]interface{}{
		paramTableFormats: []interface{}{"DELTA"},
		paramTableFormatProperties: []interface{}{map[string]interface{}{
			paramIceberg: []interface{}{map[string]interface{}{paramEnabled: false, paramFormatVersion: 2}},
			paramDelta:   []interface{}{map[string]interface{}{paramEnabled: true}},
		}},
	}
	newDisplayName := cty.StringVal("topic-new")

	tests := []struct {
		name   string
		state  map[string]interface{}
		config map[string]cty.Value
		want   string
	}{
		{
			name:  "update: a lower format_version",
			state: icebergState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(2)), testNoDelta),
			},
			want: "format_version can't be lowered from 3 to 2.",
		},
		{
			name:  "update: a changed format_version on a disabled ICEBERG",
			state: deltaState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(false, cty.NumberIntVal(3)), testDeltaList(true)),
			},
			want: "format_version can't change while ICEBERG is disabled.",
		},
		{
			name:  "update: the stored format_version on a disabled ICEBERG gives no error",
			state: deltaState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(false, cty.NumberIntVal(2)), testDeltaList(true)),
			},
		},
		{
			name:   "update: a config with only table_formats gives no error",
			state:  icebergState,
			config: map[string]cty.Value{paramTableFormats: testTableFormats("ICEBERG")},
		},
		{
			name: "create: the format_version checks are skipped",
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(false, cty.NumberIntVal(2)), testDeltaList(true)),
			},
		},
		{
			name:  "update: a lower format_version with the same Kafka cluster",
			state: withAttribute(icebergState, paramKafkaCluster, testBlockState(map[string]string{paramId: "lkc-1"})),
			config: map[string]cty.Value{
				paramKafkaCluster:          testBlock(r, paramKafkaCluster, map[string]string{paramId: "lkc-1"}),
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(2)), testNoDelta),
			},
			want: "format_version can't be lowered from 3 to 2.",
		},
		{
			name:  "replacement: a new display_name, so a lower format_version gives no error",
			state: withAttribute(icebergState, paramDisplayName, "topic-old"),
			config: map[string]cty.Value{
				paramDisplayName:           newDisplayName,
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(2)), testNoDelta),
			},
		},
	}
	// Each block change replaces the topic, so a lower format_version gives no error.
	replacements := []struct {
		name     string
		key      string
		old, new map[string]string
	}{
		{"a new Kafka cluster", paramKafkaCluster, map[string]string{paramId: "lkc-old"}, map[string]string{paramId: "lkc-new"}},
		{"a new environment", paramEnvironment, map[string]string{paramId: "env-old"}, map[string]string{paramId: "env-new"}},
		{"a new byob_aws bucket", paramByobAws,
			map[string]string{paramBucketName: "bucket-old", paramProviderIntegrationId: "cspi-1"},
			map[string]string{paramBucketName: "bucket-new", paramProviderIntegrationId: "cspi-1"}},
		{"a new azure_data_lake_storage_gen_2 container", paramAzureStorage,
			map[string]string{paramStorageAccount: "account", paramContainerName: "container-old", paramProviderIntegrationId: "cspi-1"},
			map[string]string{paramStorageAccount: "account", paramContainerName: "container-new", paramProviderIntegrationId: "cspi-1"}},
		{"a new google_cloud_storage bucket", paramGcsStorage,
			map[string]string{paramBucketName: "bucket-old", paramProviderIntegrationId: "cspi-1"},
			map[string]string{paramBucketName: "bucket-new", paramProviderIntegrationId: "cspi-1"}},
	}
	for _, replacement := range replacements {
		tests = append(tests, struct {
			name   string
			state  map[string]interface{}
			config map[string]cty.Value
			want   string
		}{
			name:  "replacement: " + replacement.name + ", so a lower format_version gives no error",
			state: withAttribute(icebergState, replacement.key, testBlockState(replacement.old)),
			config: map[string]cty.Value{
				replacement.key:            testBlock(r, replacement.key, replacement.new),
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(2)), testNoDelta),
			},
		})
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

// TestTableflowTopicTableFormatsCustomizeDiff plans with tableflowTopicTableFormatsCustomizeDiff and checks which of
// table_formats and table_format_properties are known after apply (NewComputed).
func TestTableflowTopicTableFormatsCustomizeDiff(t *testing.T) {
	r := tableflowTopicTestResource()
	r.CustomizeDiff = tableflowTopicTableFormatsCustomizeDiff

	icebergState := map[string]interface{}{
		paramTableFormats: []interface{}{"ICEBERG"},
		paramTableFormatProperties: []interface{}{map[string]interface{}{
			paramIceberg: []interface{}{map[string]interface{}{paramEnabled: true, paramFormatVersion: 3}},
			paramDelta:   []interface{}{map[string]interface{}{paramEnabled: false}},
		}},
	}

	tests := []struct {
		name   string
		state  map[string]interface{}
		config map[string]cty.Value
		want   []string
	}{
		{
			// On create, the SDK already marks an attribute that the config does not set as known after apply.
			name:   "create: table_formats stays known and table_format_properties is known after apply",
			config: map[string]cty.Value{paramTableFormats: testTableFormats("ICEBERG", "DELTA")},
			want:   []string{paramTableFormatProperties},
		},
		{
			name: "create: table_format_properties stays known and table_formats is known after apply",
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(3)), testDeltaList(true)),
			},
			want: []string{paramTableFormats},
		},
		{
			name:  "update: a DELTA enabled change without table_formats makes table_formats known after apply",
			state: icebergState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testNoIceberg, testDeltaList(true)),
			},
			want: []string{paramTableFormats},
		},
		{
			name:  "update: an ICEBERG enabled change without table_formats makes table_formats known after apply",
			state: icebergState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(false, cty.NullVal(cty.Number)), testDeltaList(true)),
			},
			want: []string{paramTableFormats},
		},
		{
			name:  "update: unchanged enabled values without table_formats make nothing known after apply",
			state: icebergState,
			config: map[string]cty.Value{
				paramTableFormatProperties: testTableFormatProperties(testIcebergList(true, cty.NumberIntVal(3)), testNoDelta),
			},
		},
		{
			name:   "update: a table_formats change without table_format_properties makes table_format_properties known after apply",
			state:  icebergState,
			config: map[string]cty.Value{paramTableFormats: testTableFormats("ICEBERG", "DELTA")},
			want:   []string{paramTableFormatProperties},
		},
		{
			name:   "update: an unchanged table_formats without table_format_properties makes nothing known after apply",
			state:  icebergState,
			config: map[string]cty.Value{paramTableFormats: testTableFormats("ICEBERG")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff, err := planTableflowTopicTestResource(t, r, tt.state, tt.config)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, name := range []string{paramTableFormatProperties, paramTableFormats} {
				if diff != nil && diff.Attributes[name+".#"] != nil && diff.Attributes[name+".#"].NewComputed {
					got = append(got, name)
				}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got known after apply %q, want %q", got, tt.want)
			}
		})
	}
}
