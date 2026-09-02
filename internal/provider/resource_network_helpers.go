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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/samber/lo"
)

// computedGatewaySchema is the Computed-only schema for the read-only spec.gateway ObjectRef.
// Referenced by name via terraform.objectref_schema_overrides in cli-terraform-generator's
// registry.yaml, because the provider's own gatewaySchema() is the *writable* form
// resource_dns_record.go uses as an input (Required, MinItems/MaxItems 1, ForceNew, Required id).
// The generator's <attr>Schema() dedup guard matches by name, so without the override
// confluent_network would bind to that one and turn a read-only attribute into a required input.
// Moved here verbatim from resource_network.go, which regeneration overwrites.
func computedGatewaySchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Computed: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				paramId: {
					Type:     schema.TypeString,
					Computed: true,
				},
			},
		},
	}
}

// networkResourceCustomizeDiff suppresses a spurious diff on zones when the old and new lists
// hold the same set of elements in a different order -- the API derives zones_info from the
// user-supplied zones list and may return it reordered. Referenced by name via
// terraform.customize_diff; kept hand-written because it is cross-field diff logic the spec
// cannot express.
//
// Renamed from the hand-written setNetworkDiff, which was wired as
// customdiff.Sequence(setNetworkDiff); with a single function that wrapper is a no-op, so the
// behavior is unchanged.
func networkResourceCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if !diff.HasChange(paramZones) {
		return nil
	}
	oldZonesInterface, newZonesInterfaces := diff.GetChange(paramZones)
	oldZones := convertToStringSlice(oldZonesInterface.([]interface{}))
	newZones := convertToStringSlice(newZonesInterfaces.([]interface{}))

	// Check whether oldZones and newZones have the same set of elements
	intersection := lo.Intersect(oldZones, newZones)
	oldAndNewZonesHaveSameSetOfElements := len(oldZones) == len(newZones) && len(oldZones) == len(intersection)

	if oldAndNewZonesHaveSameSetOfElements {
		// Set old value to paramZones to avoid TF drift
		if err := diff.SetNew(paramZones, oldZones); err != nil {
			return fmt.Errorf("error customizing diff Network: %s", createDescriptiveError(err))
		}
	}

	return nil
}
