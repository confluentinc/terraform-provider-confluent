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
	"errors"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	tableflowv1 "github.com/confluentinc/ccloud-sdk-go-v2/tableflow/v1"
)

var (
	deltaEnabledKey         = fmt.Sprintf("%s.0.%s.0.%s", paramTableFormatProperties, paramDelta, paramEnabled)
	icebergEnabledKey       = fmt.Sprintf("%s.0.%s.0.%s", paramTableFormatProperties, paramIceberg, paramEnabled)
	icebergFormatVersionKey = fmt.Sprintf("%s.0.%s.0.%s", paramTableFormatProperties, paramIceberg, paramFormatVersion)
)

func expandTableFormatPropertiesForCreate(raw cty.Value) []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf {
	var result []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf
	if iceberg := configuredIcebergProperties(raw); iceberg != nil {
		result = append(result, tableflowv1.TableflowV1IcebergTableFormatPropertiesAsTableflowV1TableflowTopicSpecTableFormatPropertiesOneOf(iceberg))
	}
	if delta := configuredDeltaProperties(raw); delta != nil {
		result = append(result, tableflowv1.TableflowV1DeltaTableFormatPropertiesAsTableflowV1TableflowTopicSpecTableFormatPropertiesOneOf(delta))
	}
	return result
}

func expandTableFormatPropertiesForUpdate(raw cty.Value) []tableflowv1.TableflowV1TableflowTopicSpecUpdateTableFormatPropertiesOneOf {
	var result []tableflowv1.TableflowV1TableflowTopicSpecUpdateTableFormatPropertiesOneOf
	if iceberg := configuredIcebergProperties(raw); iceberg != nil {
		result = append(result, tableflowv1.TableflowV1IcebergTableFormatPropertiesAsTableflowV1TableflowTopicSpecUpdateTableFormatPropertiesOneOf(iceberg))
	}
	if delta := configuredDeltaProperties(raw); delta != nil {
		result = append(result, tableflowv1.TableflowV1DeltaTableFormatPropertiesAsTableflowV1TableflowTopicSpecUpdateTableFormatPropertiesOneOf(delta))
	}
	return result
}

func flattenTableFormatProperties(props []tableflowv1.TableflowV1TableflowTopicSpecTableFormatPropertiesOneOf) []interface{} {
	block := map[string]interface{}{}
	for _, prop := range props {
		if iceberg := prop.TableflowV1IcebergTableFormatProperties; iceberg != nil {
			block[paramIceberg] = []interface{}{map[string]interface{}{
				paramEnabled:       iceberg.GetEnabled(),
				paramFormatVersion: int(iceberg.GetFormatVersion()),
			}}
		} else if delta := prop.TableflowV1DeltaTableFormatProperties; delta != nil {
			block[paramDelta] = []interface{}{map[string]interface{}{
				paramEnabled: delta.GetEnabled(),
			}}
		}
	}
	if len(block) == 0 {
		return nil
	}
	return []interface{}{block}
}

// 0 means that there is no version.
func checkFormatVersionNotLowered(oldVersion, newVersion int) error {
	if oldVersion == 0 || newVersion == 0 || newVersion >= oldVersion {
		return nil
	}
	return fmt.Errorf("format_version can't be lowered from %d to %d.", oldVersion, newVersion)
}

// checkDisabledIcebergVersion checks the format_version of a disabled ICEBERG. It can repeat the stored version, but not
// change it. 0 for newVersion means that there is no version. 0 for oldVersion means that ICEBERG never had a version,
// so any version is a change.
func checkDisabledIcebergVersion(enabled bool, oldVersion, newVersion int) error {
	if enabled || newVersion == 0 || newVersion == oldVersion {
		return nil
	}
	return errors.New("format_version can't change while ICEBERG is disabled.")
}

// tableflowTopicTableFormatPropertiesCustomizeDiff does the plan-time checks of table_format_properties: format_version
// can't be lowered, and a disabled ICEBERG can't change format_version. It stops at the first failed check. The API
// does all other checks. The checks read only the configured values, and on update the stored format_version.
func tableflowTopicTableFormatPropertiesCustomizeDiff(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
	raw := diff.GetRawConfig()
	if raw.IsNull() || !raw.IsKnown() {
		return nil
	}
	if !ctyAttr(raw, paramTableFormatProperties).IsWhollyKnown() {
		tflog.Debug(ctx, fmt.Sprintf("Skipping the plan checks of Tableflow Topic %q: %q is not known yet", diff.Id(), paramTableFormatProperties))
		return nil
	}

	iceberg := configuredIcebergProperties(raw)
	if !iceberg.HasFormatVersion() {
		return nil
	}

	// The format_version checks compare the configured version with the version in the state. So they are only for an
	// update of the same topic.
	isCreate := diff.Id() == ""
	isKafkaClusterReplacement := diff.HasChange(paramKafkaCluster)
	isEnvironmentReplacement := diff.HasChange(paramEnvironment)
	isDisplayNameReplacement := diff.HasChange(paramDisplayName)
	isStorageReplacement := diff.HasChange(paramByobAws) || diff.HasChange(paramAzureStorage) || diff.HasChange(paramGcsStorage)
	if isCreate || isKafkaClusterReplacement || isEnvironmentReplacement || isDisplayNameReplacement || isStorageReplacement {
		return nil
	}

	old, _ := diff.GetChange(icebergFormatVersionKey)
	oldVersion, _ := old.(int)
	newVersion := int(iceberg.GetFormatVersion())
	if err := checkFormatVersionNotLowered(oldVersion, newVersion); err != nil {
		return err
	}
	return checkDisabledIcebergVersion(iceberg.GetEnabled(), oldVersion, newVersion)
}

func configuredIcebergProperties(raw cty.Value) *tableflowv1.TableflowV1IcebergTableFormatProperties {
	block := configuredTableFormatBlock(raw, paramIceberg)
	enabled, ok := ctyBool(ctyAttr(block, paramEnabled))
	if !ok {
		return nil
	}
	props := tableflowv1.NewTableflowV1IcebergTableFormatProperties(tableFormatIceberg, enabled)
	if version, ok := ctyInt(ctyAttr(block, paramFormatVersion)); ok {
		props.SetFormatVersion(int32(version))
	}
	return props
}

func configuredDeltaProperties(raw cty.Value) *tableflowv1.TableflowV1DeltaTableFormatProperties {
	enabled, ok := ctyBool(ctyAttr(configuredTableFormatBlock(raw, paramDelta), paramEnabled))
	if !ok {
		return nil
	}
	return tableflowv1.NewTableflowV1DeltaTableFormatProperties(tableFormatDelta, enabled)
}

func configuredTableFormatBlock(raw cty.Value, name string) cty.Value {
	return ctyFirstBlock(ctyAttr(ctyFirstBlock(ctyAttr(raw, paramTableFormatProperties)), name))
}

// tableflowTopicTableFormatsCustomizeDiff marks table_formats as known after apply when an enabled value of
// table_format_properties changes and the configuration does not set table_formats. The API then changes
// table_formats. After apply, the SDK keeps the planned value of a set attribute that is wholly known, so without this
// the state keeps the old table_formats until the next refresh.
// In the opposite direction, it marks table_format_properties as known after apply when table_formats changes and the
// configuration does not set table_format_properties. The API then changes the enabled values, so without this the
// plan shows the old enabled values. The update request does not send table_format_properties then, because
// expandTableFormatPropertiesForUpdate returns nil for a configuration without the block.
func tableflowTopicTableFormatsCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	// On create, an attribute that the configuration does not set is already known after apply.
	if diff.Id() == "" {
		return nil
	}
	raw := diff.GetRawConfig()
	tableFormatsConfigured := !ctyAttr(raw, paramTableFormats).IsNull()
	tableFormatPropertiesConfigured := !ctyFirstBlock(ctyAttr(raw, paramTableFormatProperties)).IsNull()
	if !tableFormatsConfigured && (diff.HasChange(icebergEnabledKey) || diff.HasChange(deltaEnabledKey)) {
		return diff.SetNewComputed(paramTableFormats)
	}
	if !tableFormatPropertiesConfigured && diff.HasChange(paramTableFormats) {
		return diff.SetNewComputed(paramTableFormatProperties)
	}
	return nil
}

// ctyInt returns the value of a known number, and false for ok when v is null, unknown or not a number.
func ctyInt(v cty.Value) (int, bool) {
	if v.IsNull() || !v.IsKnown() || !v.Type().Equals(cty.Number) {
		return 0, false
	}
	value, _ := v.AsBigFloat().Int64()
	return int(value), true
}
