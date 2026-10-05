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
	"strings"

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

// networkResourceCustomizeDiff is the resource-level CustomizeDiff for confluent_network,
// referenced by name via terraform.customize_diff; kept hand-written because it is cross-field
// logic the spec cannot express.
//
// Renamed from the hand-written setNetworkDiff, which was wired as
// customdiff.Sequence(setNetworkDiff); with a single function that wrapper is a no-op, so the
// suppression behavior is unchanged.
//
// Order matters: the zones reorder is suppressed BEFORE validation runs. Until it is,
// HasChange(paramZones) reports a change on a network that is not being replaced, which would
// make validateNetworkOnCreateOrReplace re-validate existing infrastructure -- on precisely the
// drift suppressNetworkZonesReorder exists to absorb. SetNew writes through the same newWriter
// that HasChange reads, so once suppressed the change is gone for the validator too.
func networkResourceCustomizeDiff(_ context.Context, diff *schema.ResourceDiff, _ interface{}) error {
	if err := suppressNetworkZonesReorder(diff); err != nil {
		return err
	}
	return validateNetworkOnCreateOrReplace(diff)
}

// suppressNetworkZonesReorder suppresses a spurious diff on zones when the old and new lists hold
// the same set of elements in a different order -- the API derives zones_info from the
// user-supplied zones list and may return it reordered.
//
// Was the body of networkResourceCustomizeDiff; the logic is unchanged, extracted so its early
// return no longer skips validation.
func suppressNetworkZonesReorder(diff *schema.ResourceDiff) error {
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

// acceptedNetworkConnectionTypes is the closed set the hand-written resource enforced on
// connection_types. The spec's x-extensible-enum cannot be closed by the generator (the API may add
// a value, and closing it in the schema would reject a valid config at plan time), and the TF SDK
// has no ValidateFunc for list elements anyway -- the hand-written resource_network.go said as much
// in a TODO on connectionTypesSchema(). So the check lives here, where it can also be scoped to the
// create path rather than running against every existing network.
var acceptedNetworkConnectionTypes = []string{connectionTypePeering, connectionTypeTransitGateway, connectionTypePrivateLink}

// validateNetworkOnCreateOrReplace restores the two cross-field checks the hand-written
// networkCreate ran before building its request: the connection_types closed-set check
// (verifyListValues) and the zones/cloud/connection_types compatibility rule (validateZones).
// Generated CRUD does not carry them, and neither is expressible in the OpenAPI spec, so they
// belong in CustomizeDiff -- the one hook the generator hands back to the provider for exactly this.
//
// Runs when Terraform is about to CREATE a network -- which an empty Id identifies on its own, for
// both a brand-new resource and an existing one being replaced. When the first diff pass requires
// replacement, helper/schema nils out the state, re-diffs, and re-runs CustomizeDiff against that
// nil state (schema.go, the `if result.RequiresNew()` block), so the replacement plan reaches this
// func a second time with Id() == "". Testing the ForceNew fields for changes instead would be both
// redundant and wrong: on the FIRST pass an existing network still has its Id, and a zones list the
// API merely reordered reads as a change -- which would re-validate working infrastructure that the
// published provider plans cleanly. TestNetworkCustomizeDiffSkipsValidationOnZonesReorder and
// TestNetworkCustomizeDiffValidatesOnForceNewReplacement lock in both halves.
//
// The trade against the published behavior is direction, not strictness: these fire at plan instead
// of at apply, so a bad config is caught before anything is created.
func validateNetworkOnCreateOrReplace(diff *schema.ResourceDiff) error {
	if diff.Id() != "" {
		return nil
	}

	// An unknown value reads back as the zero value ("" for a string, an empty list for a list) with
	// no way to tell it apart from a real empty, so cross-checking one here rejects a valid config at
	// plan time. Terraform re-plans each resource with its values resolved immediately before applying
	// it, so these checks still run before the API call; in the worst case the API rejects the request,
	// which is what the hand-written resource did. This mirrors the SDK's own rule -- helper/schema
	// guards ConflictsWith and friends with isWhollyKnown for the same reason.
	if !diff.NewValueKnown(paramCloud) ||
		!networkDiffListKnown(diff, paramConnectionTypes) ||
		!networkDiffListKnown(diff, paramZones) {
		return nil
	}

	connectionTypes := convertToStringSlice(diff.Get(paramConnectionTypes).([]interface{}))
	if err := verifyListValues(connectionTypes, acceptedNetworkConnectionTypes, false); err != nil {
		return fmt.Errorf("input validation error reading Network's %q: %s", paramConnectionTypes, createDescriptiveError(err))
	}

	zones := convertToStringSlice(diff.Get(paramZones).([]interface{}))
	cloud := diff.Get(paramCloud).(string)
	if err := validateNetworkZones(zones, cloud, connectionTypes); err != nil {
		return fmt.Errorf("input validation error reading Network's %q: %s", paramZones, createDescriptiveError(err))
	}

	return nil
}

// networkDiffListKnown reports whether a list attribute is fully known at diff time.
//
// Both levels have to be checked. ResourceDiff.NewValueKnown on a list reflects only the element
// count: readListField propagates Computed from the "#" count and not from the elements, so a list
// of known length holding an unknown element (connection_types = [some_resource.x.y]) reports known
// while that element reads back as "".
func networkDiffListKnown(diff *schema.ResourceDiff, key string) bool {
	if !diff.NewValueKnown(key) {
		return false
	}
	for i := range diff.Get(key).([]interface{}) {
		if !diff.NewValueKnown(fmt.Sprintf("%s.%d", key, i)) {
			return false
		}
	}
	return true
}

// validateNetworkZones is the hand-written validateZones, moved out of resource_network.go before
// regeneration overwrote it. Only AWS (PrivateLink, Peering or TransitGateway) and GCP
// (Private Service Connect or Peering) networks accept a caller-supplied zone list; everywhere else
// Confluent Cloud chooses the zones.
func validateNetworkZones(zones []string, cloud string, connectionTypes []string) error {
	if len(zones) == 0 {
		return nil
	}

	awsWithZoneCapableConnection := strings.EqualFold(cloud, paramAws) &&
		(stringInSlice(connectionTypePrivateLink, connectionTypes, false) ||
			stringInSlice(connectionTypeTransitGateway, connectionTypes, false) ||
			stringInSlice(connectionTypePeering, connectionTypes, false))
	gcpWithZoneCapableConnection := strings.EqualFold(cloud, paramGcp) &&
		(stringInSlice(connectionTypePrivateLink, connectionTypes, false) ||
			stringInSlice(connectionTypePeering, connectionTypes, false))

	if !awsWithZoneCapableConnection && !gcpWithZoneCapableConnection {
		return fmt.Errorf("zones can only be specified for AWS networks used with PrivateLink or Peering or TransitGateway or for GCP networks used with Private Service Connect or Peering")
	}
	return nil
}
